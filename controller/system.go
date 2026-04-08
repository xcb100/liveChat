package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"goflylivechat/agent"
	"goflylivechat/common"
	"goflylivechat/models"
	"goflylivechat/tools"
)

// GetHealthz 输入请求上下文，输出为存活检查结果，目的在于向负载均衡与监控系统提供轻量存活信号。
func GetHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "ok",
		"result": gin.H{
			"status": "alive",
			"time":   time.Now().Format(time.RFC3339),
		},
	})
}

// GetReadyz 输入请求上下文，输出为就绪检查结果，目的在于验证数据库、缓存与 gRPC agent 链路的可用性。
func GetReadyz(c *gin.Context) {
	databaseReady := checkDatabaseReady()
	redisReady := checkRedisReady()
	agentReady := checkAgentReady(c.Request.Context())
	dispatchReady := checkAgentDispatchReady(c.Request.Context())

	tools.SetReadinessMetric("mysql", databaseReady)
	tools.SetReadinessMetric("redis", redisReady)
	tools.SetReadinessMetric("agent_grpc", agentReady)
	tools.SetReadinessMetric("agent_dispatch", dispatchReady)

	if databaseReady && redisReady && agentReady && dispatchReady {
		c.JSON(http.StatusOK, gin.H{
			"code": http.StatusOK,
			"msg":  "ok",
			"result": gin.H{
				"mysql":          databaseReady,
				"redis":          redisReady,
				"agent_grpc":     agentReady,
				"agent_dispatch": dispatchReady,
			},
		})
		return
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{
		"code": http.StatusServiceUnavailable,
		"msg":  "service not ready",
		"result": gin.H{
			"mysql":          databaseReady,
			"redis":          redisReady,
			"agent_grpc":     agentReady,
			"agent_dispatch": dispatchReady,
		},
	})
}

// GetVersion 输入请求上下文，输出为服务版本与构建信息，目的在于为运维和部署校验提供稳定元数据端点。
func GetVersion(c *gin.Context) {
	appConfig := common.GetAppConfig()
	agentState := "unavailable"
	dispatchMode := "disabled"
	dispatchReady := false
	agentClient := agent.GetDefaultClient()
	if agentClient != nil {
		agentState = agentClient.State().String()
	}
	agentDispatcher := agent.GetDefaultDispatcher()
	if agentDispatcher != nil {
		dispatchMode = agentDispatcher.Mode()
		dispatchReady = agentDispatcher.CheckHealth(c.Request.Context()) == nil
	}
	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "ok",
		"result": gin.H{
			"service_name":         appConfig.ServiceName,
			"version":              appConfig.BuildVersion,
			"commit":               appConfig.BuildCommit,
			"build_time":           appConfig.BuildTime,
			"agent_state":          agentState,
			"agent_dispatch_mode":  dispatchMode,
			"agent_dispatch_ready": dispatchReady,
			"time":                 time.Now().Format(time.RFC3339),
		},
	})
}

// GetAgentStatus 输入请求上下文，输出为 agent 列表与容量信息，目的在于为工作台和运维检查提供智能客服状态视图。
func GetAgentStatus(c *gin.Context) {
	agentClient := agent.GetDefaultClient()
	if agentClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"code": http.StatusOK,
			"msg":  "ok",
			"result": gin.H{
				"agents": []gin.H{},
			},
		})
		return
	}

	agentDescriptors, listError := agentClient.ListAgents(c.Request.Context(), false, "")
	if listError != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  listError.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "ok",
		"result": gin.H{
			"agents": agentDescriptors,
		},
	})
}

// checkDatabaseReady 输入为空，输出为数据库就绪状态，目的在于快速判断 MySQL 链路是否可用。
func checkDatabaseReady() bool {
	return models.IsSetupReady()
}

// checkRedisReady 输入为空，输出为 Redis 就绪状态，目的在于快速判断缓存链路是否可用。
func checkRedisReady() bool {
	redisClient := tools.GetRedis()
	if redisClient == nil {
		return false
	}
	return redisClient.Ping(tools.Ctx).Err() == nil
}

// checkAgentReady 输入请求上下文，输出为 agent gRPC 就绪状态，目的在于验证 agent 服务链路是否可用。
func checkAgentReady(ctx context.Context) bool {
	agentClient := agent.GetDefaultClient()
	if agentClient == nil {
		return false
	}
	return agentClient.CheckHealth(ctx) == nil
}

// checkAgentDispatchReady 输入请求上下文，输出为 agent 调度链路就绪状态，目的在于在启用 Kafka 解耦后额外验证消息投递通道。
func checkAgentDispatchReady(ctx context.Context) bool {
	agentDispatcher := agent.GetDefaultDispatcher()
	if agentDispatcher == nil {
		return false
	}
	return agentDispatcher.CheckHealth(ctx) == nil
}
