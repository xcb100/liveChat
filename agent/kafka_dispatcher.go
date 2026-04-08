package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"goflylivechat/common"
)

const (
	sessionCommandAssign  = "assign"
	sessionCommandRelease = "release"
)

type sessionCommand struct {
	Action      string    `json:"action"`
	VisitorID   string    `json:"visitor_id"`
	VisitorName string    `json:"visitor_name,omitempty"`
	Capability  string    `json:"capability,omitempty"`
	Source      string    `json:"source,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type kafkaMessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type kafkaMessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type brokerDialContext func(ctx context.Context, network string, address string) (net.Conn, error)

type KafkaDispatcher struct {
	client         sessionInvoker
	writer         kafkaMessageWriter
	reader         kafkaMessageReader
	brokers        []string
	dialBroker     brokerDialContext
	onAssigned     func(visitorID string, displayName string)
	enqueueTimeout time.Duration
	processTimeout time.Duration
	retryBackoff   time.Duration
	consumerCancel context.CancelFunc
	consumerDone   chan struct{}
}

// NewKafkaDispatcher 输入 agent 客户端、应用配置和回调钩子，输出为 Kafka 调度器实例，目的在于把 agent 分配与释放动作从主请求链路中解耦出来。
func NewKafkaDispatcher(client *Client, appConfig common.AppConfig, hooks DispatcherHooks) (*KafkaDispatcher, error) {
	if client == nil {
		return nil, ErrDispatcherUnavailable
	}
	if len(appConfig.AgentKafkaBrokers) == 0 {
		return nil, errors.New("LIVECHAT_AGENT_KAFKA_BROKERS is required when dispatch mode is kafka")
	}
	if strings.TrimSpace(appConfig.AgentKafkaTopic) == "" {
		return nil, errors.New("LIVECHAT_AGENT_KAFKA_TOPIC is required when dispatch mode is kafka")
	}

	dialer := &kafka.Dialer{
		Timeout:  appConfig.AgentKafkaDialTimeout,
		ClientID: appConfig.AgentKafkaClientID,
	}
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(appConfig.AgentKafkaBrokers...),
		Topic:                  appConfig.AgentKafkaTopic,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           10 * time.Millisecond,
		AllowAutoTopicCreation: true,
		WriteTimeout:           appConfig.AgentKafkaWriteTimeout,
		ReadTimeout:            appConfig.AgentKafkaReadTimeout,
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               appConfig.AgentKafkaBrokers,
		Topic:                 appConfig.AgentKafkaTopic,
		GroupID:               appConfig.AgentKafkaGroupID,
		Dialer:                dialer,
		MinBytes:              1,
		MaxBytes:              1 << 20,
		MaxWait:               appConfig.AgentKafkaReadTimeout,
		QueueCapacity:         64,
		ReadLagInterval:       -1,
		WatchPartitionChanges: true,
	})

	return newKafkaDispatcherWithIO(client, appConfig, writer, reader, hooks, func(ctx context.Context, network string, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, address)
	}, true), nil
}

func newKafkaDispatcherWithIO(client sessionInvoker, appConfig common.AppConfig, writer kafkaMessageWriter, reader kafkaMessageReader, hooks DispatcherHooks, dialBroker brokerDialContext, startConsumer bool) *KafkaDispatcher {
	dispatcher := &KafkaDispatcher{
		client:         client,
		writer:         writer,
		reader:         reader,
		brokers:        append([]string(nil), appConfig.AgentKafkaBrokers...),
		dialBroker:     dialBroker,
		onAssigned:     hooks.OnAssigned,
		enqueueTimeout: appConfig.AgentKafkaEnqueueTimeout,
		processTimeout: appConfig.AgentRequestTimeout,
		retryBackoff:   appConfig.AgentKafkaConsumeBackoff,
		consumerDone:   make(chan struct{}),
	}

	if !startConsumer {
		close(dispatcher.consumerDone)
		return dispatcher
	}

	consumerContext, cancel := context.WithCancel(context.Background())
	dispatcher.consumerCancel = cancel
	go dispatcher.consumeLoop(consumerContext)
	return dispatcher
}

// AssignSession 输入上下文和分配请求，输出为投递结果，目的在于把 agent 分配请求异步写入 Kafka。
func (dispatcher *KafkaDispatcher) AssignSession(ctx context.Context, request AssignRequest) error {
	return dispatcher.publish(ctx, sessionCommand{
		Action:      sessionCommandAssign,
		VisitorID:   request.VisitorID,
		VisitorName: request.VisitorName,
		Capability:  request.Capability,
		Source:      request.Source,
		CreatedAt:   time.Now(),
	})
}

// ReleaseSession 输入上下文和访客标识，输出为投递结果，目的在于把 agent 释放请求异步写入 Kafka。
func (dispatcher *KafkaDispatcher) ReleaseSession(ctx context.Context, visitorID string) error {
	return dispatcher.publish(ctx, sessionCommand{
		Action:    sessionCommandRelease,
		VisitorID: visitorID,
		CreatedAt: time.Now(),
	})
}

// CheckHealth 输入请求上下文，输出为健康检查结果，目的在于同时验证 Kafka broker 与 agent gRPC 链路是否可用。
func (dispatcher *KafkaDispatcher) CheckHealth(ctx context.Context) error {
	if dispatcher == nil || dispatcher.client == nil {
		return net.ErrClosed
	}
	if checkError := dispatcher.client.CheckHealth(ctx); checkError != nil {
		return checkError
	}
	if len(dispatcher.brokers) == 0 || dispatcher.dialBroker == nil {
		return errors.New("kafka broker dialer is not configured")
	}

	checkContext, cancel := context.WithTimeout(ctx, dispatcher.enqueueTimeout)
	defer cancel()
	connection, dialError := dispatcher.dialBroker(checkContext, "tcp", dispatcher.brokers[0])
	if dialError != nil {
		return dialError
	}
	return connection.Close()
}

// Mode 输入为空，输出为调度模式标识，目的在于对外暴露 Kafka 解耦已启用的事实。
func (dispatcher *KafkaDispatcher) Mode() string {
	_ = dispatcher
	return DispatchModeKafka
}

// Close 输入为空，输出为资源关闭结果，目的在于停止 Kafka 消费循环并释放读写句柄。
func (dispatcher *KafkaDispatcher) Close() error {
	if dispatcher == nil {
		return nil
	}
	if dispatcher.consumerCancel != nil {
		dispatcher.consumerCancel()
	}

	var closeError error
	if dispatcher.reader != nil {
		closeError = dispatcher.reader.Close()
	}
	if dispatcher.writer != nil {
		writerCloseError := dispatcher.writer.Close()
		if closeError == nil {
			closeError = writerCloseError
		}
	}
	if dispatcher.consumerDone != nil {
		<-dispatcher.consumerDone
	}
	return closeError
}

func (dispatcher *KafkaDispatcher) publish(ctx context.Context, command sessionCommand) error {
	if dispatcher == nil || dispatcher.writer == nil {
		return ErrDispatcherUnavailable
	}
	commandBytes, marshalError := json.Marshal(command)
	if marshalError != nil {
		return marshalError
	}

	publishContext, cancel := context.WithTimeout(ctx, dispatcher.enqueueTimeout)
	defer cancel()

	return dispatcher.writer.WriteMessages(publishContext, kafka.Message{
		Key:   []byte(command.VisitorID),
		Value: commandBytes,
		Time:  command.CreatedAt,
	})
}

func (dispatcher *KafkaDispatcher) consumeLoop(ctx context.Context) {
	defer close(dispatcher.consumerDone)

	for {
		message, fetchError := dispatcher.reader.FetchMessage(ctx)
		if fetchError != nil {
			if ctx.Err() != nil || errors.Is(fetchError, context.Canceled) {
				return
			}
			log.Printf("agent kafka dispatcher fetch failed: %v", fetchError)
			dispatcher.sleepBackoff(ctx)
			continue
		}

		shouldCommit, handleError := dispatcher.handleMessage(message)
		if handleError != nil {
			log.Printf("agent kafka dispatcher handle failed: %v", handleError)
		}
		if !shouldCommit {
			dispatcher.sleepBackoff(ctx)
			continue
		}
		if commitError := dispatcher.reader.CommitMessages(ctx, message); commitError != nil {
			if ctx.Err() != nil || errors.Is(commitError, context.Canceled) {
				return
			}
			log.Printf("agent kafka dispatcher commit failed: %v", commitError)
			dispatcher.sleepBackoff(ctx)
		}
	}
}

func (dispatcher *KafkaDispatcher) handleMessage(message kafka.Message) (bool, error) {
	var command sessionCommand
	if unmarshalError := json.Unmarshal(message.Value, &command); unmarshalError != nil {
		return true, fmt.Errorf("decode session command: %w", unmarshalError)
	}
	if strings.TrimSpace(command.VisitorID) == "" {
		return true, errors.New("session command visitor_id is required")
	}

	callContext, cancel := context.WithTimeout(context.Background(), dispatcher.processTimeout)
	defer cancel()

	switch command.Action {
	case sessionCommandAssign:
		assignResponse, assignError := dispatcher.client.AssignSession(callContext, command.VisitorID, command.VisitorName, command.Capability, command.Source)
		if assignError != nil {
			return false, assignError
		}
		if assignResponse != nil && assignResponse.GetAssigned() && dispatcher.onAssigned != nil {
			dispatcher.onAssigned(command.VisitorID, assignResponse.GetDisplayName())
		}
		return true, nil
	case sessionCommandRelease:
		_, releaseError := dispatcher.client.ReleaseSession(callContext, command.VisitorID)
		if releaseError != nil {
			return false, releaseError
		}
		return true, nil
	default:
		return true, fmt.Errorf("unsupported session command action: %s", command.Action)
	}
}

func (dispatcher *KafkaDispatcher) sleepBackoff(ctx context.Context) {
	timer := time.NewTimer(dispatcher.retryBackoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
