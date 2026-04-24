package controller

import (
	"github.com/gin-gonic/gin"
	"goflylivechat/models"
)

func GetRoleList(c *gin.Context) {
	roles := models.FindRoles()
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "获取成功",
		"result": roles,
	})
}
func PostRole(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": 400,
		"msg":  "P0阶段暂不支持在线修改角色定义",
	})
}
