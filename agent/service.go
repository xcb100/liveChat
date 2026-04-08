package agent

import (
	"context"
	"errors"

	"goflylivechat/agentpb"
)

type Service struct {
	agentpb.UnimplementedAgentServiceServer
	registry *Registry
}

// NewService 输入注册中心实例，输出为 gRPC agent 服务实现，目的在于对外暴露 agent 注册、分配与释放能力。
func NewService(registry *Registry) *Service {
	return &Service{registry: registry}
}

// RegisterAgent 输入注册请求，输出为注册结果，目的在于让外部 agent 客户端加入调度池。
func (service *Service) RegisterAgent(ctx context.Context, request *agentpb.RegisterAgentRequest) (*agentpb.RegisterAgentResponse, error) {
	_ = ctx
	if request.GetAgent() == nil {
		return nil, errors.New("agent registration is required")
	}
	runtimeAgent := service.registry.Register(
		request.GetAgent().GetAgentId(),
		request.GetAgent().GetDisplayName(),
		request.GetAgent().GetCapabilities(),
		request.GetAgent().GetMaxSessions(),
		request.GetAgent().GetEnabled(),
	)
	return &agentpb.RegisterAgentResponse{Agent: buildAgentDescriptor(runtimeAgent)}, nil
}

// Heartbeat 输入心跳请求，输出为最新 agent 视图，目的在于刷新 agent 存活时间。
func (service *Service) Heartbeat(ctx context.Context, request *agentpb.HeartbeatRequest) (*agentpb.HeartbeatResponse, error) {
	_ = ctx
	runtimeAgent, exists := service.registry.Heartbeat(request.GetAgentId())
	if !exists {
		return nil, errors.New("agent not found")
	}
	return &agentpb.HeartbeatResponse{Agent: buildAgentDescriptor(runtimeAgent)}, nil
}

// ListAgents 输入查询请求，输出为 agent 列表，目的在于提供 agent 容量与状态查询能力。
func (service *Service) ListAgents(ctx context.Context, request *agentpb.ListAgentsRequest) (*agentpb.ListAgentsResponse, error) {
	_ = ctx
	runtimeAgents := service.registry.List(request.GetOnlyAvailable(), request.GetCapability())
	agentDescriptors := make([]*agentpb.AgentDescriptor, 0, len(runtimeAgents))
	for _, runtimeAgent := range runtimeAgents {
		agentDescriptors = append(agentDescriptors, buildAgentDescriptor(runtimeAgent))
	}
	return &agentpb.ListAgentsResponse{Agents: agentDescriptors}, nil
}

// AssignSession 输入分配请求，输出为分配结果，目的在于为待接待访客分配一个可用 agent。
func (service *Service) AssignSession(ctx context.Context, request *agentpb.AssignSessionRequest) (*agentpb.AssignSessionResponse, error) {
	_ = ctx
	runtimeAgent, reason, assigned := service.registry.Assign(request.GetVisitorId(), request.GetVisitorName(), request.GetPreferredCapability())
	return &agentpb.AssignSessionResponse{
		Assigned:    assigned,
		AgentId:     runtimeAgent.AgentID,
		DisplayName: runtimeAgent.DisplayName,
		Reason:      reason,
	}, nil
}

// ReleaseSession 输入释放请求，输出为释放结果，目的在于在会话结束后释放 agent 占用。
func (service *Service) ReleaseSession(ctx context.Context, request *agentpb.ReleaseSessionRequest) (*agentpb.ReleaseSessionResponse, error) {
	_ = ctx
	return &agentpb.ReleaseSessionResponse{
		Released: service.registry.Release(request.GetVisitorId()),
	}, nil
}

// buildAgentDescriptor 输入运行时 agent 视图，输出为 protobuf 描述对象，目的在于统一 gRPC 响应结构。
func buildAgentDescriptor(runtimeAgent RuntimeAgent) *agentpb.AgentDescriptor {
	return &agentpb.AgentDescriptor{
		AgentId:           runtimeAgent.AgentID,
		DisplayName:       runtimeAgent.DisplayName,
		Capabilities:      runtimeAgent.Capabilities,
		MaxSessions:       runtimeAgent.MaxSessions,
		ActiveSessions:    uint32(len(runtimeAgent.ActiveSessions)),
		AvailableSessions: runtimeAgent.AvailableSessions(),
		Available:         runtimeAgent.isAvailable(),
		Enabled:           runtimeAgent.Enabled,
		UpdatedAtUnix:     runtimeAgent.UpdatedAt.Unix(),
	}
}
