package agent

import (
	"testing"
	"time"
)

// TestRegistryAssignAndRelease 输入测试上下文，输出为调度断言结果，目的在于验证 agent 容量分配和释放行为符合预期。
func TestRegistryAssignAndRelease(t *testing.T) {
	registry := NewRegistry(30 * time.Second)
	registry.Register("agent-a", "Agent A", []string{"chat"}, 1, true)
	registry.Register("agent-b", "Agent B", []string{"chat", "sales"}, 2, true)

	assignedAgent, reason, assigned := registry.Assign("visitor-1", "Visitor 1", "chat")
	if !assigned {
		t.Fatalf("期望成功分配 visitor-1，失败原因: %s", reason)
	}
	if assignedAgent.AgentID == "" {
		t.Fatalf("期望得到有效 agent 标识")
	}

	_, reason, assigned = registry.Assign("visitor-2", "Visitor 2", "support")
	if assigned {
		t.Fatalf("期望 capability 不匹配时分配失败")
	}
	if reason == "" {
		t.Fatalf("期望 capability 不匹配时返回失败原因")
	}

	released := registry.Release("visitor-1")
	if !released {
		t.Fatalf("期望成功释放 visitor-1")
	}
}
