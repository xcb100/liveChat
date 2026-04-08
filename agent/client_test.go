package agent

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"goflylivechat/agentpb"
	"goflylivechat/common"
	"goflylivechat/tools"
)

const clientTestBufferSize = 1024 * 1024

type slowAgentService struct {
	agentpb.UnimplementedAgentServiceServer
}

// ListAgents 输入上下文和查询请求，输出为延迟后的空列表响应，目的在于模拟超时场景下的 gRPC 服务端行为。
func (service *slowAgentService) ListAgents(ctx context.Context, request *agentpb.ListAgentsRequest) (*agentpb.ListAgentsResponse, error) {
	time.Sleep(120 * time.Millisecond)
	return &agentpb.ListAgentsResponse{}, nil
}

// TestClientListAgentsTimeout 输入测试上下文，输出为超时断言结果，目的在于验证 agent 客户端请求超时控制生效。
func TestClientListAgentsTimeout(t *testing.T) {
	listener := bufconn.Listen(clientTestBufferSize)
	grpcServer := grpc.NewServer()
	agentpb.RegisterAgentServiceServer(grpcServer, &slowAgentService{})
	go func() {
		if serveError := grpcServer.Serve(listener); serveError != nil {
			t.Errorf("启动慢速 gRPC 测试服务失败: %v", serveError)
		}
	}()
	defer grpcServer.Stop()

	connection, dialError := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithUnaryInterceptor(tools.UnaryClientLoggingInterceptor()),
	)
	if dialError != nil {
		t.Fatalf("连接 gRPC 测试服务失败: %v", dialError)
	}
	defer connection.Close()

	appConfig := common.GetAppConfig()
	appConfig.AgentRequestTimeout = 40 * time.Millisecond
	client := newClientWithConnection(connection, appConfig)

	_, listError := client.ListAgents(context.Background(), false, "")
	if listError == nil {
		t.Fatalf("期望 ListAgents 超时失败")
	}
	if !strings.Contains(listError.Error(), "deadline") {
		t.Fatalf("期望得到 deadline 相关错误，实际为 %v", listError)
	}
}
