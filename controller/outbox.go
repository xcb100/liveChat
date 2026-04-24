package controller

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"goflylivechat/common"
	"goflylivechat/models"
)

func GetOutboxEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pagesize", strconv.Itoa(int(common.PageSize))))
	if pageSize <= 0 {
		pageSize = int(common.PageSize)
	}

	outboxEvents := models.FindOutboxEvents(uint(page), uint(pageSize))
	totalCount := models.CountOutboxEvents()

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": gin.H{
			"list":     outboxEvents,
			"count":    totalCount,
			"page":     page,
			"pagesize": pageSize,
		},
	})
}
