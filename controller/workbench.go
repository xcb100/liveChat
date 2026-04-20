package controller

import (
	"sort"

	"github.com/gin-gonic/gin"
	"goflylivechat/agent"
	"goflylivechat/common"
	"goflylivechat/models"
	"goflylivechat/routing"
	"goflylivechat/ws"
)

type workbenchBootstrapResult struct {
	Profile          gin.H       `json:"profile"`
	AssignedSessions []gin.H     `json:"assigned_sessions"`
	PendingSessions  []gin.H     `json:"pending_sessions"`
	OnlineVisitors   []gin.H     `json:"online_visitors"`
	RecentVisitors   gin.H       `json:"recent_visitors"`
	ReplyGroups      interface{} `json:"reply_groups"`
	Blacklists       interface{} `json:"blacklists"`
	KefuOverview     []gin.H     `json:"kefu_overview"`
	AgentOverview    []gin.H     `json:"agent_overview"`
	Metrics          gin.H       `json:"metrics"`
}

// GetWorkbenchBootstrap 输入请求上下文，输出为工作台初始化数据，目的在于减少前端首屏多次请求。
func GetWorkbenchBootstrap(c *gin.Context) {
	kefuNameValue, _ := c.Get("kefu_name")
	kefuName := kefuNameValue.(string)
	user := models.FindUser(kefuName)

	recentVisitors := models.FindVisitorsByKefuId(1, common.VisitorPageSize, kefuName)
	recentVisitorCount := models.CountVisitorsByKefuId(kefuName)
	replyGroups := models.FindReplyByUserId(kefuName)
	blacklists := models.FindIpsByKefuId(kefuName)
	assignedSessions := buildWorkbenchSessionSummaries(routing.SessionListFilter{OwnerID: kefuName, RouteStatus: routing.RouteStatusAssigned})
	pendingSessions := buildWorkbenchSessionSummaries(routing.SessionListFilter{RouteStatus: routing.RouteStatusPending})
	onlineVisitors := buildWorkbenchOnlineVisitors(kefuName)
	kefuOverview, totalKefus, availableKefus := loadKefuOverview()
	agentOverview, totalAgents, availableAgents := loadAgentOverview(c)

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": workbenchBootstrapResult{
			Profile: gin.H{
				"uid":                user.ID,
				"username":           user.Name,
				"nickname":           user.Nickname,
				"avator":             user.Avator,
				"routing_skills":     models.FindConfigByUserId(kefuName, "RoutingSkills").ConfValue,
				"presence_status":    models.GetUserKefuPresenceStatus(kefuName),
				"accepting_sessions": models.GetUserKefuAcceptingSessions(kefuName),
			},
			AssignedSessions: assignedSessions,
			PendingSessions:  pendingSessions,
			OnlineVisitors:   onlineVisitors,
			RecentVisitors: gin.H{
				"list":     recentVisitors,
				"count":    recentVisitorCount,
				"page":     1,
				"pagesize": common.VisitorPageSize,
			},
			ReplyGroups:   replyGroups,
			Blacklists:    blacklists,
			KefuOverview:  kefuOverview,
			AgentOverview: agentOverview,
			Metrics: gin.H{
				"assigned_count":        len(assignedSessions),
				"pending_count":         len(pendingSessions),
				"online_count":          len(onlineVisitors),
				"recent_count":          recentVisitorCount,
				"reply_count":           len(replyGroups),
				"blacklist_count":       len(blacklists),
				"kefu_total_count":      totalKefus,
				"kefu_available_count":  availableKefus,
				"agent_total_count":     totalAgents,
				"agent_available_count": availableAgents,
			},
		},
	})
}

// buildWorkbenchOnlineVisitors 输入客服账号，输出为在线访客摘要列表，目的在于为工作台构建稳定排序的在线会话数据。
func buildWorkbenchOnlineVisitors(kefuName string) []gin.H {
	onlineVisitors := make([]gin.H, 0)
	visitorIDs := make([]string, 0)
	for visitorID, visitor := range ws.ClientList {
		if visitor.To_id != kefuName {
			continue
		}
		onlineVisitors = append(onlineVisitors, gin.H{
			"uid":        visitorID,
			"username":   visitor.Name,
			"avator":     visitor.Avator,
			"updated_at": visitor.UpdateTime.Unix(),
		})
		visitorIDs = append(visitorIDs, visitorID)
	}

	lastMessageMap := make(map[string]string, len(visitorIDs))
	for _, message := range models.FindLastMessage(visitorIDs) {
		lastMessageMap[message.VisitorId] = message.Content
	}

	for _, visitor := range onlineVisitors {
		visitorID := visitor["uid"].(string)
		lastMessage := lastMessageMap[visitorID]
		if lastMessage == "" {
			lastMessage = "new visitor"
		}
		visitor["last_message"] = lastMessage
	}

	sort.SliceStable(onlineVisitors, func(leftIndex, rightIndex int) bool {
		return onlineVisitors[leftIndex]["updated_at"].(int64) > onlineVisitors[rightIndex]["updated_at"].(int64)
	})

	return onlineVisitors
}

func buildWorkbenchSessionSummaries(filter routing.SessionListFilter) []gin.H {
	sessionSnapshots := routing.GetDefaultCenter().ListSessions(filter)
	if len(sessionSnapshots) == 0 {
		return []gin.H{}
	}

	visitorIDs := make([]string, 0, len(sessionSnapshots))
	for _, sessionSnapshot := range sessionSnapshots {
		visitorIDs = append(visitorIDs, sessionSnapshot.VisitorID)
	}

	visitorMap := make(map[string]models.Visitor, len(visitorIDs))
	for _, visitor := range models.FindVisitorsByVisitorIds(visitorIDs) {
		visitorMap[visitor.VisitorId] = visitor
	}

	lastMessageMap := make(map[string]string, len(visitorIDs))
	for _, message := range models.FindLastMessage(visitorIDs) {
		lastMessageMap[message.VisitorId] = message.Content
	}

	sessionSummaries := make([]gin.H, 0, len(sessionSnapshots))
	for _, sessionSnapshot := range sessionSnapshots {
		visitor := visitorMap[sessionSnapshot.VisitorID]
		displayName := sessionSnapshot.VisitorName
		if displayName == "" {
			displayName = visitor.Name
		}
		if displayName == "" {
			displayName = "访客"
		}
		avator := visitor.Avator
		if avator == "" {
			avator = "/static/images/2.png"
		}
		lastMessage := lastMessageMap[sessionSnapshot.VisitorID]
		if lastMessage == "" {
			lastMessage = visitor.LastMessage
		}
		if lastMessage == "" {
			lastMessage = "new visitor"
		}
		sessionSummaries = append(sessionSummaries, gin.H{
			"uid":                 sessionSnapshot.VisitorID,
			"visitor_id":          sessionSnapshot.VisitorID,
			"username":            displayName,
			"name":                displayName,
			"avator":              avator,
			"owner_id":            sessionSnapshot.OwnerID,
			"sticky_owner_id":     sessionSnapshot.StickyOwnerID,
			"route_status":        sessionSnapshot.RouteStatus,
			"queue_name":          sessionSnapshot.QueueName,
			"last_route_reason":   sessionSnapshot.LastRouteReason,
			"queue_entered_at":    sessionSnapshot.QueueEnteredAt.Unix(),
			"last_assign_attempt": sessionSnapshot.LastAssignAttemptAt.Unix(),
			"updated_at":          sessionSnapshot.LastActivityAt.Unix(),
			"status":              visitor.Status,
			"last_message":        lastMessage,
			"preferred_skill":     sessionSnapshot.PreferredSkill,
		})
	}

	return sessionSummaries
}

func loadKefuOverview() ([]gin.H, int, int) {
	runtimeKefus := routing.GetDefaultCenter().ListKefus()
	kefuOverview := make([]gin.H, 0, len(runtimeKefus))
	availableKefus := 0
	for _, runtimeKefu := range runtimeKefus {
		if runtimeKefu.PresenceStatus == routing.PresenceOnline && runtimeKefu.AcceptingSessions && runtimeKefu.ActiveSessions < runtimeKefu.MaxSessions {
			availableKefus++
		}
		kefuOverview = append(kefuOverview, gin.H{
			"kefu_id":            runtimeKefu.KefuID,
			"display_name":       runtimeKefu.DisplayName,
			"skills":             runtimeKefu.Skills,
			"presence_status":    runtimeKefu.PresenceStatus,
			"accepting_sessions": runtimeKefu.AcceptingSessions,
			"active_sessions":    runtimeKefu.ActiveSessions,
			"max_sessions":       runtimeKefu.MaxSessions,
		})
	}
	return kefuOverview, len(runtimeKefus), availableKefus
}

// loadAgentOverview 输入请求上下文，输出为 agent 摘要、总数与可用数，目的在于把智能客服待命能力汇总给工作台。
func loadAgentOverview(c *gin.Context) ([]gin.H, int, int) {
	agentClient := agent.GetDefaultClient()
	if agentClient == nil {
		return []gin.H{}, 0, 0
	}

	agentDescriptors, listError := agentClient.ListAgents(c.Request.Context(), false, "")
	if listError != nil {
		return []gin.H{}, 0, 0
	}

	agentOverview := make([]gin.H, 0, len(agentDescriptors))
	availableAgents := 0
	for _, agentDescriptor := range agentDescriptors {
		if agentDescriptor.GetAvailable() {
			availableAgents++
		}
		agentOverview = append(agentOverview, gin.H{
			"agent_id":           agentDescriptor.GetAgentId(),
			"display_name":       agentDescriptor.GetDisplayName(),
			"capabilities":       agentDescriptor.GetCapabilities(),
			"available":          agentDescriptor.GetAvailable(),
			"active_sessions":    agentDescriptor.GetActiveSessions(),
			"available_sessions": agentDescriptor.GetAvailableSessions(),
		})
	}

	return agentOverview, len(agentDescriptors), availableAgents
}
