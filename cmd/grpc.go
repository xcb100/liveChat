package cmd

import (
	"log"
	"net"

	"google.golang.org/grpc"
	grpcHealth "google.golang.org/grpc/health"
	grpcHealthV1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"goflylivechat/agent"
	"goflylivechat/agentpb"
	"goflylivechat/common"
	"goflylivechat/tools"
)

// mustStartGRPCServer 输入应用配置，输出为 gRPC 服务与监听器，目的在于启动 agent 调度所需的 gRPC 基础能力。
func mustStartGRPCServer(appConfig common.AppConfig) (*grpc.Server, net.Listener) {
	listener, listenError := net.Listen("tcp", "0.0.0.0:"+appConfig.GRPCPort)
	if listenError != nil {
		log.Fatalf("gRPC 监听失败: %v", listenError)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tools.UnaryServerLoggingInterceptor(),
		),
	)
	agentpb.RegisterAgentServiceServer(grpcServer, agent.NewService(agent.GetDefaultRegistry()))

	healthServer := grpcHealth.NewServer()
	healthServer.SetServingStatus("", grpcHealthV1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("goflylivechat.agent.v1.AgentService", grpcHealthV1.HealthCheckResponse_SERVING)
	grpcHealthV1.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	go func() {
		if serveError := grpcServer.Serve(listener); serveError != nil {
			log.Fatalf("gRPC 服务启动失败: %v", serveError)
		}
	}()

	return grpcServer, listener
}
