package agent

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type RuntimeAgent struct {
	AgentID        string
	DisplayName    string
	Capabilities   []string
	MaxSessions    uint32
	ActiveSessions map[string]time.Time
	Enabled        bool
	UpdatedAt      time.Time
}

type Registry struct {
	heartbeatTTL time.Duration
	lock         sync.RWMutex
	agents       map[string]*RuntimeAgent
	sessionIndex map[string]string
}

var defaultRegistry = NewRegistry(75 * time.Second)

// NewRegistry 输入心跳 TTL，输出为 agent 注册中心实例，目的在于统一管理 agent 生命周期与会话分配。
func NewRegistry(heartbeatTTL time.Duration) *Registry {
	return &Registry{
		heartbeatTTL: heartbeatTTL,
		agents:       make(map[string]*RuntimeAgent),
		sessionIndex: make(map[string]string),
	}
}

// ConfigureDefaultRegistry 输入心跳 TTL，输出为默认注册中心重建结果，目的在于按运行配置初始化全局 agent 注册中心。
func ConfigureDefaultRegistry(heartbeatTTL time.Duration) {
	defaultRegistry = NewRegistry(heartbeatTTL)
}

// GetDefaultRegistry 输入为空，输出为默认注册中心实例，目的在于为 gRPC 服务与本地调度共享 agent 状态。
func GetDefaultRegistry() *Registry {
	return defaultRegistry
}

// Register 输入 agent 基本信息，输出为运行时 agent 视图，目的在于注册或更新一个智能客服代理。
func (registry *Registry) Register(agentID string, displayName string, capabilities []string, maxSessions uint32, enabled bool) RuntimeAgent {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	runtimeAgent, exists := registry.agents[agentID]
	if !exists {
		runtimeAgent = &RuntimeAgent{
			AgentID:        agentID,
			ActiveSessions: make(map[string]time.Time),
		}
		registry.agents[agentID] = runtimeAgent
	}

	runtimeAgent.DisplayName = displayName
	runtimeAgent.Capabilities = normalizeCapabilities(capabilities)
	runtimeAgent.MaxSessions = maxSessions
	runtimeAgent.Enabled = enabled
	runtimeAgent.UpdatedAt = time.Now()

	return cloneRuntimeAgent(runtimeAgent)
}

// Heartbeat 输入 agent 标识，输出为运行时 agent 视图，目的在于刷新 agent 活跃时间并维持可调度状态。
func (registry *Registry) Heartbeat(agentID string) (RuntimeAgent, bool) {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	runtimeAgent, exists := registry.agents[agentID]
	if !exists {
		return RuntimeAgent{}, false
	}
	runtimeAgent.UpdatedAt = time.Now()
	return cloneRuntimeAgent(runtimeAgent), true
}

// List 输入是否仅返回可分配 agent 与能力筛选项，输出为 agent 列表，目的在于向工作台与调度方提供 agent 视图。
func (registry *Registry) List(onlyAvailable bool, capability string) []RuntimeAgent {
	registry.cleanupExpiredLocked()
	registry.lock.RLock()
	defer registry.lock.RUnlock()

	normalizedCapability := strings.ToLower(strings.TrimSpace(capability))
	runtimeAgents := make([]RuntimeAgent, 0, len(registry.agents))
	for _, runtimeAgent := range registry.agents {
		if normalizedCapability != "" && !runtimeAgent.hasCapability(normalizedCapability) {
			continue
		}
		if onlyAvailable && !runtimeAgent.isAvailable() {
			continue
		}
		runtimeAgents = append(runtimeAgents, cloneRuntimeAgent(runtimeAgent))
	}

	sort.SliceStable(runtimeAgents, func(leftIndex, rightIndex int) bool {
		leftAvailable := runtimeAgents[leftIndex].AvailableSessions()
		rightAvailable := runtimeAgents[rightIndex].AvailableSessions()
		if leftAvailable == rightAvailable {
			return runtimeAgents[leftIndex].UpdatedAt.After(runtimeAgents[rightIndex].UpdatedAt)
		}
		return leftAvailable > rightAvailable
	})
	return runtimeAgents
}

// Assign 输入访客标识、访客名称与偏好能力，输出为分配结果，目的在于为无人值守场景挑选一个可用 agent。
func (registry *Registry) Assign(visitorID string, visitorName string, capability string) (RuntimeAgent, string, bool) {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	registry.cleanupExpiredLocked()
	if existingAgentID, exists := registry.sessionIndex[visitorID]; exists {
		runtimeAgent, hasAgent := registry.agents[existingAgentID]
		if hasAgent {
			return cloneRuntimeAgent(runtimeAgent), "", true
		}
		delete(registry.sessionIndex, visitorID)
	}

	normalizedCapability := strings.ToLower(strings.TrimSpace(capability))
	var selectedAgent *RuntimeAgent
	for _, runtimeAgent := range registry.agents {
		if normalizedCapability != "" && !runtimeAgent.hasCapability(normalizedCapability) {
			continue
		}
		if !runtimeAgent.isAvailable() {
			continue
		}
		if selectedAgent == nil || runtimeAgent.AvailableSessions() > selectedAgent.AvailableSessions() || runtimeAgent.UpdatedAt.After(selectedAgent.UpdatedAt) {
			selectedAgent = runtimeAgent
		}
	}

	if selectedAgent == nil {
		return RuntimeAgent{}, "暂无可用 agent 客服", false
	}

	selectedAgent.ActiveSessions[visitorID] = time.Now()
	selectedAgent.UpdatedAt = time.Now()
	registry.sessionIndex[visitorID] = selectedAgent.AgentID
	_ = visitorName
	return cloneRuntimeAgent(selectedAgent), "", true
}

// Release 输入访客标识，输出为释放结果，目的在于在会话结束后归还 agent 容量。
func (registry *Registry) Release(visitorID string) bool {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	agentID, exists := registry.sessionIndex[visitorID]
	if !exists {
		return false
	}
	delete(registry.sessionIndex, visitorID)
	runtimeAgent, hasAgent := registry.agents[agentID]
	if !hasAgent {
		return false
	}
	delete(runtimeAgent.ActiveSessions, visitorID)
	runtimeAgent.UpdatedAt = time.Now()
	return true
}

// Snapshot 输入为空，输出为 agent 聚合信息，目的在于为工作台统计和健康检查提供简化状态。
func (registry *Registry) Snapshot() (int, int) {
	runtimeAgents := registry.List(false, "")
	totalAgents := len(runtimeAgents)
	availableAgents := 0
	for _, runtimeAgent := range runtimeAgents {
		if runtimeAgent.isAvailable() {
			availableAgents++
		}
	}
	return totalAgents, availableAgents
}

// AvailableSessions 输入为空，输出为可接待会话数，目的在于计算当前 agent 的剩余容量。
func (runtimeAgent RuntimeAgent) AvailableSessions() uint32 {
	if runtimeAgent.MaxSessions <= uint32(len(runtimeAgent.ActiveSessions)) {
		return 0
	}
	return runtimeAgent.MaxSessions - uint32(len(runtimeAgent.ActiveSessions))
}

// cleanupExpiredLocked 输入为空，输出为过期 agent 清理结果，目的在于移除长时间失联的 agent。
func (registry *Registry) cleanupExpiredLocked() {
	now := time.Now()
	for agentID, runtimeAgent := range registry.agents {
		if now.Sub(runtimeAgent.UpdatedAt) <= registry.heartbeatTTL {
			continue
		}
		for visitorID := range runtimeAgent.ActiveSessions {
			delete(registry.sessionIndex, visitorID)
		}
		delete(registry.agents, agentID)
	}
}

// isAvailable 输入为空，输出为 agent 是否可分配，目的在于统一判断 agent 可用性。
func (runtimeAgent RuntimeAgent) isAvailable() bool {
	return runtimeAgent.Enabled && runtimeAgent.AvailableSessions() > 0
}

// hasCapability 输入能力关键字，输出为是否支持该能力，目的在于支持按业务能力筛选 agent。
func (runtimeAgent RuntimeAgent) hasCapability(capability string) bool {
	if capability == "" {
		return true
	}
	for _, currentCapability := range runtimeAgent.Capabilities {
		if strings.EqualFold(currentCapability, capability) {
			return true
		}
	}
	return false
}

// normalizeCapabilities 输入能力列表，输出为归一化后的能力列表，目的在于清理重复与空值能力。
func normalizeCapabilities(capabilities []string) []string {
	normalizedCapabilities := make([]string, 0, len(capabilities))
	seenCapabilities := make(map[string]struct{}, len(capabilities))
	for _, capability := range capabilities {
		normalizedCapability := strings.ToLower(strings.TrimSpace(capability))
		if normalizedCapability == "" {
			continue
		}
		if _, exists := seenCapabilities[normalizedCapability]; exists {
			continue
		}
		seenCapabilities[normalizedCapability] = struct{}{}
		normalizedCapabilities = append(normalizedCapabilities, normalizedCapability)
	}
	sort.Strings(normalizedCapabilities)
	return normalizedCapabilities
}

// cloneRuntimeAgent 输入运行时 agent 指针，输出为安全副本，目的在于避免外部修改注册中心内部状态。
func cloneRuntimeAgent(runtimeAgent *RuntimeAgent) RuntimeAgent {
	clonedSessions := make(map[string]time.Time, len(runtimeAgent.ActiveSessions))
	for visitorID, assignedAt := range runtimeAgent.ActiveSessions {
		clonedSessions[visitorID] = assignedAt
	}
	clonedCapabilities := append([]string(nil), runtimeAgent.Capabilities...)
	return RuntimeAgent{
		AgentID:        runtimeAgent.AgentID,
		DisplayName:    runtimeAgent.DisplayName,
		Capabilities:   clonedCapabilities,
		MaxSessions:    runtimeAgent.MaxSessions,
		ActiveSessions: clonedSessions,
		Enabled:        runtimeAgent.Enabled,
		UpdatedAt:      runtimeAgent.UpdatedAt,
	}
}
