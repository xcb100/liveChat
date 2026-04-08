package agent

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"goflylivechat/agentpb"
	"goflylivechat/common"
)

type fakeSessionInvoker struct {
	checkHealthError error
	assignResponse   *agentpb.AssignSessionResponse
	assignError      error
	releaseResponse  *agentpb.ReleaseSessionResponse
	releaseError     error

	assignRequests  []AssignRequest
	releaseRequests []string
	mu              sync.Mutex
}

func (invoker *fakeSessionInvoker) CheckHealth(ctx context.Context) error {
	_ = ctx
	return invoker.checkHealthError
}

func (invoker *fakeSessionInvoker) AssignSession(ctx context.Context, visitorID string, visitorName string, capability string, source string) (*agentpb.AssignSessionResponse, error) {
	_ = ctx
	invoker.mu.Lock()
	invoker.assignRequests = append(invoker.assignRequests, AssignRequest{
		VisitorID:   visitorID,
		VisitorName: visitorName,
		Capability:  capability,
		Source:      source,
	})
	invoker.mu.Unlock()
	if invoker.assignResponse == nil {
		invoker.assignResponse = &agentpb.AssignSessionResponse{}
	}
	return invoker.assignResponse, invoker.assignError
}

func (invoker *fakeSessionInvoker) ReleaseSession(ctx context.Context, visitorID string) (*agentpb.ReleaseSessionResponse, error) {
	_ = ctx
	invoker.mu.Lock()
	invoker.releaseRequests = append(invoker.releaseRequests, visitorID)
	invoker.mu.Unlock()
	if invoker.releaseResponse == nil {
		invoker.releaseResponse = &agentpb.ReleaseSessionResponse{Released: true}
	}
	return invoker.releaseResponse, invoker.releaseError
}

type fakeKafkaWriter struct {
	writeError error
	messages   []kafka.Message
	mu         sync.Mutex
}

func (writer *fakeKafkaWriter) WriteMessages(ctx context.Context, messages ...kafka.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	writer.mu.Lock()
	writer.messages = append(writer.messages, messages...)
	writer.mu.Unlock()
	return writer.writeError
}

func (writer *fakeKafkaWriter) Close() error {
	return nil
}

type fakeKafkaReader struct {
	fetchChannel  chan kafka.Message
	commitChannel chan []kafka.Message
	closeChannel  chan struct{}
}

func newFakeKafkaReader() *fakeKafkaReader {
	return &fakeKafkaReader{
		fetchChannel:  make(chan kafka.Message, 1),
		commitChannel: make(chan []kafka.Message, 1),
		closeChannel:  make(chan struct{}),
	}
}

func (reader *fakeKafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	case <-reader.closeChannel:
		return kafka.Message{}, context.Canceled
	case message := <-reader.fetchChannel:
		return message, nil
	}
}

func (reader *fakeKafkaReader) CommitMessages(ctx context.Context, messages ...kafka.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case reader.commitChannel <- messages:
		return nil
	}
}

func (reader *fakeKafkaReader) Close() error {
	select {
	case <-reader.closeChannel:
	default:
		close(reader.closeChannel)
	}
	return nil
}

// TestDirectDispatcherAssignSession 输入测试上下文，输出为直连调度断言结果，目的在于验证同步模式仍会调用 gRPC 客户端并触发访客通知回调。
func TestDirectDispatcherAssignSession(t *testing.T) {
	fakeClient := &fakeSessionInvoker{
		assignResponse: &agentpb.AssignSessionResponse{
			Assigned:    true,
			DisplayName: "AI-Agent",
		},
	}

	assignedVisitorID := ""
	assignedDisplayName := ""
	dispatcher := NewDirectDispatcher(fakeClient, DispatcherHooks{
		OnAssigned: func(visitorID string, displayName string) {
			assignedVisitorID = visitorID
			assignedDisplayName = displayName
		},
	})

	assignError := dispatcher.AssignSession(context.Background(), AssignRequest{
		VisitorID:   "visitor-1",
		VisitorName: "Alice",
		Capability:  "chat",
		Source:      "visitor_auto_reply",
	})
	if assignError != nil {
		t.Fatalf("期望直连调度成功，实际报错: %v", assignError)
	}
	if len(fakeClient.assignRequests) != 1 {
		t.Fatalf("期望记录 1 次分配请求，实际为 %d", len(fakeClient.assignRequests))
	}
	if assignedVisitorID != "visitor-1" || assignedDisplayName != "AI-Agent" {
		t.Fatalf("期望触发分配通知，实际 visitor=%s display=%s", assignedVisitorID, assignedDisplayName)
	}
}

// TestKafkaDispatcherAssignPublishesCommand 输入测试上下文，输出为 Kafka 投递断言结果，目的在于验证异步模式会先把请求写入消息队列。
func TestKafkaDispatcherAssignPublishesCommand(t *testing.T) {
	fakeClient := &fakeSessionInvoker{}
	fakeWriter := &fakeKafkaWriter{}
	fakeReader := newFakeKafkaReader()
	appConfig := common.GetAppConfig()

	dispatcher := newKafkaDispatcherWithIO(fakeClient, appConfig, fakeWriter, fakeReader, DispatcherHooks{}, func(ctx context.Context, network string, address string) (net.Conn, error) {
		_ = ctx
		_ = network
		_ = address
		return nil, nil
	}, false)

	assignError := dispatcher.AssignSession(context.Background(), AssignRequest{
		VisitorID:   "visitor-1",
		VisitorName: "Alice",
		Capability:  "chat",
		Source:      "visitor_auto_reply",
	})
	if assignError != nil {
		t.Fatalf("期望 Kafka 投递成功，实际报错: %v", assignError)
	}
	if len(fakeWriter.messages) != 1 {
		t.Fatalf("期望写入 1 条 Kafka 消息，实际为 %d", len(fakeWriter.messages))
	}
	if string(fakeWriter.messages[0].Key) != "visitor-1" {
		t.Fatalf("期望 Kafka key 为 visitor-1，实际为 %s", string(fakeWriter.messages[0].Key))
	}
}

// TestKafkaDispatcherConsumesAssignAndCommits 输入测试上下文，输出为消费断言结果，目的在于验证 Kafka 模式会异步调用 agent 服务并在成功后提交消息。
func TestKafkaDispatcherConsumesAssignAndCommits(t *testing.T) {
	fakeClient := &fakeSessionInvoker{
		assignResponse: &agentpb.AssignSessionResponse{
			Assigned:    true,
			DisplayName: "AI-Agent",
		},
	}
	fakeWriter := &fakeKafkaWriter{}
	fakeReader := newFakeKafkaReader()
	appConfig := common.GetAppConfig()
	appConfig.AgentRequestTimeout = 50 * time.Millisecond
	appConfig.AgentKafkaConsumeBackoff = 5 * time.Millisecond

	assignedSignal := make(chan string, 1)
	dispatcher := newKafkaDispatcherWithIO(fakeClient, appConfig, fakeWriter, fakeReader, DispatcherHooks{
		OnAssigned: func(visitorID string, displayName string) {
			assignedSignal <- visitorID + ":" + displayName
		},
	}, func(ctx context.Context, network string, address string) (net.Conn, error) {
		_ = ctx
		_ = network
		_ = address
		return nil, nil
	}, true)
	defer func() {
		_ = dispatcher.Close()
	}()

	assignError := dispatcher.AssignSession(context.Background(), AssignRequest{
		VisitorID:   "visitor-2",
		VisitorName: "Bob",
		Capability:  "chat",
		Source:      "visitor_auto_reply",
	})
	if assignError != nil {
		t.Fatalf("期望 Kafka 投递成功，实际报错: %v", assignError)
	}
	if len(fakeWriter.messages) != 1 {
		t.Fatalf("期望写入 1 条 Kafka 消息，实际为 %d", len(fakeWriter.messages))
	}

	fakeReader.fetchChannel <- fakeWriter.messages[0]

	select {
	case assigned := <-assignedSignal:
		if assigned != "visitor-2:AI-Agent" {
			t.Fatalf("期望收到 visitor-2:AI-Agent，实际为 %s", assigned)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("等待 Kafka 消费回调超时")
	}

	select {
	case committed := <-fakeReader.commitChannel:
		if len(committed) != 1 || string(committed[0].Key) != "visitor-2" {
			t.Fatalf("期望提交 visitor-2 对应消息，实际为 %+v", committed)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("等待 Kafka 提交超时")
	}
}
