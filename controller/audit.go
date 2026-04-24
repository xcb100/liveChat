package controller

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"goflylivechat/common"
	"goflylivechat/models"
)

func RecordAuditLog(c *gin.Context, action string, targetType string, targetID string, before interface{}, after interface{}) {
	if c == nil {
		return
	}

	actorNameValue, _ := c.Get("kefu_name")
	roleNameValue, _ := c.Get("role_name")
	actorIDValue, _ := c.Get("kefu_id")
	requestIDValue, _ := c.Get("request_id")

	var actorUserID uint
	switch typedValue := actorIDValue.(type) {
	case float64:
		actorUserID = uint(typedValue)
	case uint:
		actorUserID = typedValue
	}

	beforeData := marshalAuditPayload(before)
	afterData := marshalAuditPayload(after)

	models.CreateAuditLog(&models.AuditLog{
		ActorUserID: actorUserID,
		ActorName:   stringifyAuditValue(actorNameValue),
		ActorRole:   stringifyAuditValue(roleNameValue),
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		BeforeData:  beforeData,
		AfterData:   afterData,
		ClientIP:    c.ClientIP(),
		RequestID:   stringifyAuditValue(requestIDValue),
	})
}

func GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pagesize", strconv.Itoa(int(common.PageSize))))
	if pageSize <= 0 {
		pageSize = int(common.PageSize)
	}

	auditLogs := models.FindAuditLogs(uint(page), uint(pageSize))
	totalCount := models.CountAuditLogs()

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": gin.H{
			"list":     auditLogs,
			"count":    totalCount,
			"page":     page,
			"pagesize": pageSize,
		},
	})
}

func marshalAuditPayload(payload interface{}) string {
	if payload == nil {
		return ""
	}
	payloadBytes, marshalError := json.Marshal(payload)
	if marshalError != nil {
		return stringifyAuditValue(payload)
	}
	return string(payloadBytes)
}

func stringifyAuditValue(value interface{}) string {
	switch typedValue := value.(type) {
	case nil:
		return ""
	case string:
		return typedValue
	case []byte:
		return string(typedValue)
	default:
		return fmt.Sprint(typedValue)
	}
}
