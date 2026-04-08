package tools_test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"goflylivechat/agent"
	"goflylivechat/agentpb"
)

const grpcBufferSize = 1024 * 1024

// TestAgentGRPCRegisterAndAssign 输入测试上下文，输出为 gRPC 断言结果，目的在于验证 agent 注册与会话分配链路可用。
func TestAgentGRPCRegisterAndAssign(t *testing.T) {
	listener := bufconn.Listen(grpcBufferSize)
	registry := agent.NewRegistry(30 * time.Second)
	grpcServer := grpc.NewServer()
	agentpb.RegisterAgentServiceServer(grpcServer, agent.NewService(registry))
	go func() {
		if serveError := grpcServer.Serve(listener); serveError != nil {
			t.Errorf("启动 gRPC 测试服务失败: %v", serveError)
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
	)
	if dialError != nil {
		t.Fatalf("连接 gRPC 测试服务失败: %v", dialError)
	}
	defer connection.Close()

	client := agentpb.NewAgentServiceClient(connection)
	registerResponse, registerError := client.RegisterAgent(context.Background(), &agentpb.RegisterAgentRequest{
		Agent: &agentpb.AgentRegistration{
			AgentId:      "agent-test",
			DisplayName:  "Agent Test",
			Capabilities: []string{"chat", "sales"},
			MaxSessions:  2,
			Enabled:      true,
		},
	})
	if registerError != nil {
		t.Fatalf("注册 agent 失败: %v", registerError)
	}
	if registerResponse.GetAgent().GetAgentId() != "agent-test" {
		t.Fatalf("期望注册 agent-test，实际得到 %s", registerResponse.GetAgent().GetAgentId())
	}

	assignResponse, assignError := client.AssignSession(context.Background(), &agentpb.AssignSessionRequest{
		VisitorId:           "visitor-001",
		VisitorName:         "Visitor 001",
		PreferredCapability: "chat",
		Source:              "test",
	})
	if assignError != nil {
		t.Fatalf("分配会话失败: %v", assignError)
	}
	if !assignResponse.GetAssigned() {
		t.Fatalf("期望成功分配会话，实际原因: %s", assignResponse.GetReason())
	}
	if assignResponse.GetAgentId() != "agent-test" {
		t.Fatalf("期望分配给 agent-test，实际得到 %s", assignResponse.GetAgentId())
	}
}
