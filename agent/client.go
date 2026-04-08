package agent

import (
	"context"
	"net"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	grpcHealth "google.golang.org/grpc/health/grpc_health_v1"

	"goflylivechat/agentpb"
	"goflylivechat/common"
	"goflylivechat/tools"
)

type Client struct {
	connection     *grpc.ClientConn
	agentClient    agentpb.AgentServiceClient
	healthClient   grpcHealth.HealthClient
	breaker        *gobreaker.CircuitBreaker
	requestTimeout time.Duration
}

var defaultClient *Client

// NewClient 输入目标地址和应用配置，输出为 gRPC agent 客户端，目的在于通过带熔断的调用访问 agent 服务。
func NewClient(address string, appConfig common.AppConfig) (*Client, error) {
	dialContext, cancel := context.WithTimeout(context.Background(), appConfig.AgentDialTimeout)
	defer cancel()

	connection, dialError := grpc.DialContext(
		dialContext,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithUnaryInterceptor(tools.UnaryClientLoggingInterceptor()),
	)
	if dialError != nil {
		return nil, dialError
	}

	return newClientWithConnection(connection, appConfig), nil
}

// newClientWithConnection 输入 gRPC 连接和应用配置，输出为 agent 客户端，目的在于复用客户端初始化逻辑并支持测试注入。
func newClientWithConnection(connection *grpc.ClientConn, appConfig common.AppConfig) *Client {
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "agent-grpc-client",
		Timeout:     appConfig.CircuitBreakerTimeout,
		MaxRequests: appConfig.CircuitBreakerMaxHalf,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	return &Client{
		connection:     connection,
		agentClient:    agentpb.NewAgentServiceClient(connection),
		healthClient:   grpcHealth.NewHealthClient(connection),
		breaker:        breaker,
		requestTimeout: appConfig.AgentRequestTimeout,
	}
}

// SetDefaultClient 输入 agent 客户端实例，输出为默认客户端设置结果，目的在于让 HTTP 与 WebSocket 层复用同一个 gRPC 调用入口。
func SetDefaultClient(client *Client) {
	defaultClient = client
}

// GetDefaultClient 输入为空，输出为默认 agent 客户端，目的在于为业务层提供全局 gRPC agent 访问入口。
func GetDefaultClient() *Client {
	return defaultClient
}

// Close 输入为空，输出为连接关闭结果，目的在于在应用退出时释放 gRPC 客户端连接。
func (client *Client) Close() error {
	if client == nil || client.connection == nil {
		return nil
	}
	return client.connection.Close()
}

// CheckHealth 输入请求上下文，输出为健康检查结果，目的在于验证 gRPC agent 服务链路可用性。
func (client *Client) CheckHealth(ctx context.Context) error {
	if client == nil {
		return net.ErrClosed
	}
	checkContext, cancel := context.WithTimeout(ctx, client.requestTimeout)
	defer cancel()
	_, healthError := client.healthClient.Check(checkContext, &grpcHealth.HealthCheckRequest{})
	return healthError
}

// ListAgents 输入上下文、是否仅返回可用 agent 与能力筛选项，输出为 protobuf agent 列表，目的在于查询 agent 容量。
func (client *Client) ListAgents(ctx context.Context, onlyAvailable bool, capability string) ([]*agentpb.AgentDescriptor, error) {
	response, invokeError := client.execute(ctx, func(callContext context.Context) (interface{}, error) {
		return client.agentClient.ListAgents(callContext, &agentpb.ListAgentsRequest{
			OnlyAvailable: onlyAvailable,
			Capability:    capability,
		})
	})
	if invokeError != nil {
		return nil, invokeError
	}
	return response.(*agentpb.ListAgentsResponse).GetAgents(), nil
}

// AssignSession 输入上下文与会话分配请求，输出为分配结果，目的在于在无人值守时尝试转给 agent 客服。
func (client *Client) AssignSession(ctx context.Context, visitorID string, visitorName string, capability string, source string) (*agentpb.AssignSessionResponse, error) {
	response, invokeError := client.execute(ctx, func(callContext context.Context) (interface{}, error) {
		return client.agentClient.AssignSession(callContext, &agentpb.AssignSessionRequest{
			VisitorId:           visitorID,
			VisitorName:         visitorName,
			PreferredCapability: capability,
			Source:              source,
		})
	})
	if invokeError != nil {
		return nil, invokeError
	}
	return response.(*agentpb.AssignSessionResponse), nil
}

// ReleaseSession 输入上下文与访客标识，输出为释放结果，目的在于在会话结束时归还 agent 容量。
func (client *Client) ReleaseSession(ctx context.Context, visitorID string) (*agentpb.ReleaseSessionResponse, error) {
	response, invokeError := client.execute(ctx, func(callContext context.Context) (interface{}, error) {
		return client.agentClient.ReleaseSession(callContext, &agentpb.ReleaseSessionRequest{
			VisitorId: visitorID,
		})
	})
	if invokeError != nil {
		return nil, invokeError
	}
	return response.(*agentpb.ReleaseSessionResponse), nil
}

// State 输入为空，输出为 gRPC 连接状态，目的在于为健康检查和调试提供链路状态信息。
func (client *Client) State() connectivity.State {
	if client == nil || client.connection == nil {
		return connectivity.Shutdown
	}
	return client.connection.GetState()
}

// execute 输入上下文和 gRPC 调用函数，输出为调用结果，目的在于统一附加超时控制与熔断保护。
func (client *Client) execute(ctx context.Context, invoker func(context.Context) (interface{}, error)) (interface{}, error) {
	callContext, cancel := context.WithTimeout(ctx, client.requestTimeout)
	defer cancel()
	return client.breaker.Execute(func() (interface{}, error) {
		return invoker(callContext)
	})
}
