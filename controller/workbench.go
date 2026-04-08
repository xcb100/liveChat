package controller

import (
	"sort"

	"github.com/gin-gonic/gin"
	"goflylivechat/agent"
	"goflylivechat/common"
	"goflylivechat/models"
	"goflylivechat/ws"
)

type workbenchBootstrapResult struct {
	Profile        gin.H       `json:"profile"`
	OnlineVisitors []gin.H     `json:"online_visitors"`
	RecentVisitors gin.H       `json:"recent_visitors"`
	ReplyGroups    interface{} `json:"reply_groups"`
	Blacklists     interface{} `json:"blacklists"`
	AgentOverview  []gin.H     `json:"agent_overview"`
	Metrics        gin.H       `json:"metrics"`
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
	onlineVisitors := buildWorkbenchOnlineVisitors(kefuName)
	agentOverview, totalAgents, availableAgents := loadAgentOverview(c)

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": workbenchBootstrapResult{
			Profile: gin.H{
				"uid":      user.ID,
				"username": user.Name,
				"nickname": user.Nickname,
				"avator":   user.Avator,
			},
			OnlineVisitors: onlineVisitors,
			RecentVisitors: gin.H{
				"list":     recentVisitors,
				"count":    recentVisitorCount,
				"page":     1,
				"pagesize": common.VisitorPageSize,
			},
			ReplyGroups:   replyGroups,
			Blacklists:    blacklists,
			AgentOverview: agentOverview,
			Metrics: gin.H{
				"online_count":          len(onlineVisitors),
				"recent_count":          recentVisitorCount,
				"reply_count":           len(replyGroups),
				"blacklist_count":       len(blacklists),
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
