package agent

import (
	"context"
	"errors"
	"net"
	"strings"

	"goflylivechat/agentpb"
	"goflylivechat/common"
)

const (
	DispatchModeDirect = "direct"
	DispatchModeKafka  = "kafka"
)

var (
	defaultDispatcher        Dispatcher
	ErrDispatcherUnavailable = errors.New("agent dispatcher is unavailable")
)

type AssignRequest struct {
	VisitorID   string
	VisitorName string
	Capability  string
	Source      string
}

type DispatcherHooks struct {
	OnAssigned func(visitorID string, displayName string)
}

type Dispatcher interface {
	AssignSession(ctx context.Context, request AssignRequest) error
	ReleaseSession(ctx context.Context, visitorID string) error
	CheckHealth(ctx context.Context) error
	Mode() string
	Close() error
}

type sessionInvoker interface {
	CheckHealth(ctx context.Context) error
	AssignSession(ctx context.Context, visitorID string, visitorName string, capability string, source string) (*agentpb.AssignSessionResponse, error)
	ReleaseSession(ctx context.Context, visitorID string) (*agentpb.ReleaseSessionResponse, error)
}

type DirectDispatcher struct {
	client     sessionInvoker
	onAssigned func(visitorID string, displayName string)
}

// NewDispatcher 输入 agent 客户端、应用配置和回调钩子，输出为调度器实例，目的在于按配置装配同步或 Kafka 解耦的 agent 调用链路。
func NewDispatcher(client *Client, appConfig common.AppConfig, hooks DispatcherHooks) (Dispatcher, error) {
	switch normalizeDispatchMode(appConfig.AgentDispatchMode) {
	case DispatchModeKafka:
		return NewKafkaDispatcher(client, appConfig, hooks)
	default:
		return NewDirectDispatcher(client, hooks), nil
	}
}

// NewDirectDispatcher 输入 agent 客户端和回调钩子，输出为直连调度器，目的在于保留当前同步 gRPC 调用行为。
func NewDirectDispatcher(client sessionInvoker, hooks DispatcherHooks) *DirectDispatcher {
	return &DirectDispatcher{
		client:     client,
		onAssigned: hooks.OnAssigned,
	}
}

// SetDefaultDispatcher 输入调度器实例，输出为默认调度器设置结果，目的在于让 WebSocket 与 HTTP 层共享同一条 agent 调度入口。
func SetDefaultDispatcher(dispatcher Dispatcher) {
	defaultDispatcher = dispatcher
}

// GetDefaultDispatcher 输入为空，输出为默认调度器实例，目的在于向业务层暴露统一的 agent 调度方式。
func GetDefaultDispatcher() Dispatcher {
	return defaultDispatcher
}

// AssignSession 输入上下文和分配请求，输出为分配执行结果，目的在于通过默认直连模式完成 agent 会话分配。
func (dispatcher *DirectDispatcher) AssignSession(ctx context.Context, request AssignRequest) error {
	if dispatcher == nil || dispatcher.client == nil {
		return ErrDispatcherUnavailable
	}
	assignResponse, assignError := dispatcher.client.AssignSession(ctx, request.VisitorID, request.VisitorName, request.Capability, request.Source)
	if assignError != nil {
		return assignError
	}
	if assignResponse != nil && assignResponse.GetAssigned() && dispatcher.onAssigned != nil {
		dispatcher.onAssigned(request.VisitorID, assignResponse.GetDisplayName())
	}
	return nil
}

// ReleaseSession 输入上下文和访客标识，输出为释放执行结果，目的在于通过默认直连模式归还 agent 会话占用。
func (dispatcher *DirectDispatcher) ReleaseSession(ctx context.Context, visitorID string) error {
	if dispatcher == nil || dispatcher.client == nil {
		return ErrDispatcherUnavailable
	}
	_, releaseError := dispatcher.client.ReleaseSession(ctx, visitorID)
	return releaseError
}

// CheckHealth 输入请求上下文，输出为调度器健康检查结果，目的在于让直连模式复用底层 gRPC 健康状态。
func (dispatcher *DirectDispatcher) CheckHealth(ctx context.Context) error {
	if dispatcher == nil || dispatcher.client == nil {
		return net.ErrClosed
	}
	return dispatcher.client.CheckHealth(ctx)
}

// Mode 输入为空，输出为调度模式标识，目的在于为监控、状态接口和日志提供稳定模式名。
func (dispatcher *DirectDispatcher) Mode() string {
	_ = dispatcher
	return DispatchModeDirect
}

// Close 输入为空，输出为关闭结果，目的在于满足统一调度器接口并兼容优雅停机流程。
func (dispatcher *DirectDispatcher) Close() error {
	_ = dispatcher
	return nil
}

func normalizeDispatchMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case DispatchModeKafka:
		return DispatchModeKafka
	default:
		return DispatchModeDirect
	}
}
