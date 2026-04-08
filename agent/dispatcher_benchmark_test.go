package agent

import (
	"context"
	"net"
	"testing"
	"time"

	"goflylivechat/agentpb"
	"goflylivechat/common"
)

// BenchmarkDirectDispatcherAssignSession 输入基准测试上下文，输出为同步调度性能结果，目的在于给 Kafka 解耦前后的调用成本提供直连基线。
func BenchmarkDirectDispatcherAssignSession(b *testing.B) {
	fakeClient := &fakeSessionInvoker{
		assignResponse: &agentpb.AssignSessionResponse{
			Assigned:    true,
			DisplayName: "AI-Agent",
		},
	}
	dispatcher := NewDirectDispatcher(fakeClient, DispatcherHooks{})
	request := AssignRequest{
		VisitorID:   "visitor-bench",
		VisitorName: "Bench",
		Capability:  "chat",
		Source:      "benchmark",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if assignError := dispatcher.AssignSession(context.Background(), request); assignError != nil {
			b.Fatalf("直连调度失败: %v", assignError)
		}
	}
}

// BenchmarkKafkaDispatcherAssignSession 输入基准测试上下文，输出为异步投递性能结果，目的在于衡量把主路径改为 Kafka 入队后的请求侧成本。
func BenchmarkKafkaDispatcherAssignSession(b *testing.B) {
	fakeClient := &fakeSessionInvoker{}
	fakeWriter := &fakeKafkaWriter{}
	fakeReader := newFakeKafkaReader()
	appConfig := common.GetAppConfig()
	appConfig.AgentKafkaEnqueueTimeout = time.Second

	dispatcher := newKafkaDispatcherWithIO(fakeClient, appConfig, fakeWriter, fakeReader, DispatcherHooks{}, func(ctx context.Context, network string, address string) (net.Conn, error) {
		_ = ctx
		_ = network
		_ = address
		return nil, nil
	}, false)
	request := AssignRequest{
		VisitorID:   "visitor-bench",
		VisitorName: "Bench",
		Capability:  "chat",
		Source:      "benchmark",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if assignError := dispatcher.AssignSession(context.Background(), request); assignError != nil {
			b.Fatalf("Kafka 投递失败: %v", assignError)
		}
	}
}
