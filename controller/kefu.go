package controller

import (
	"github.com/gin-gonic/gin"
	"goflylivechat/middleware"
	"goflylivechat/models"
	"goflylivechat/outbox"
	"goflylivechat/routing"
	"goflylivechat/tools"
	"goflylivechat/ws"
	"net/http"
	"strconv"
)

func PostKefuAvator(c *gin.Context) {

	avator := c.PostForm("avator")
	if avator == "" {
		c.JSON(200, gin.H{
			"code":   400,
			"msg":    "不能为空",
			"result": "",
		})
		return
	}
	kefuName, _ := c.Get("kefu_name")
	models.UpdateUserAvator(kefuName.(string), avator)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": "",
	})
}
func PostKefuPass(c *gin.Context) {
	kefuName, _ := c.Get("kefu_name")
	newPass := c.PostForm("new_pass")
	confirmNewPass := c.PostForm("confirm_new_pass")
	old_pass := c.PostForm("old_pass")
	if newPass != confirmNewPass {
		c.JSON(200, gin.H{
			"code":   400,
			"msg":    "密码不一致",
			"result": "",
		})
		return
	}
	user := models.FindUser(kefuName.(string))
	if !tools.VerifyPassword(user.Password, old_pass) {
		c.JSON(200, gin.H{
			"code":   400,
			"msg":    "旧密码不正确",
			"result": "",
		})
		return
	}
	hashedPassword, hashError := tools.HashPassword(newPass)
	if hashError != nil {
		c.JSON(200, gin.H{
			"code":   500,
			"msg":    "密码处理失败",
			"result": "",
		})
		return
	}
	models.UpdateUserPass(kefuName.(string), hashedPassword)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": "",
	})
}
func PostKefuClient(c *gin.Context) {
	kefuName, _ := c.Get("kefu_name")
	clientId := c.PostForm("client_id")

	if clientId == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "client_id不能为空",
		})
		return
	}
	models.CreateUserClient(kefuName.(string), clientId)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": "",
	})
}
func GetKefuInfo(c *gin.Context) {
	kefuName, _ := c.Get("kefu_name")
	user := models.FindUser(kefuName.(string))
	info := make(map[string]interface{})
	info["avator"] = user.Avator
	info["username"] = user.Name
	info["nickname"] = user.Nickname
	info["uid"] = user.ID
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": info,
	})
}
func GetKefuInfoAll(c *gin.Context) {
	id, _ := c.Get("kefu_id")
	userinfo := models.FindUserRole("user.avator,user.name,user.id, role.name role_name", id)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "验证成功",
		"result": userinfo,
	})
}
func GetOtherKefuList(c *gin.Context) {
	idStr, _ := c.Get("kefu_id")
	id := idStr.(float64)
	result := make([]interface{}, 0)
	ws.SendPingToKefuClient()
	kefus := models.FindUsers()
	for _, kefu := range kefus {
		if uint(id) == kefu.ID {
			continue
		}

		item := make(map[string]interface{})
		item["name"] = kefu.Name
		item["nickname"] = kefu.Nickname
		item["avator"] = kefu.Avator
		item["status"] = "offline"
		kefu, ok := ws.KefuList[kefu.Name]
		if ok && kefu != nil {
			item["status"] = "online"
		}
		result = append(result, item)
	}
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": result,
	})
}
func PostTransKefu(c *gin.Context) {
	kefuId := c.Query("kefu_id")
	visitorId := c.Query("visitor_id")
	user := models.FindUser(kefuId)
	visitor := models.FindVisitorByVistorId(visitorId)
	if user.Name == "" || visitor.Name == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "访客或客服不存在",
		})
		return
	}
	sessionSnapshot, hasSession := routing.GetDefaultCenter().GetSession(visitorId)
	routeStatus := ""
	ownerID := visitor.ToId
	if hasSession {
		routeStatus = sessionSnapshot.RouteStatus
		if sessionSnapshot.OwnerID != "" {
			ownerID = sessionSnapshot.OwnerID
		}
	}
	if !middleware.CanAccessSession(c, ownerID, routeStatus) {
		c.JSON(200, gin.H{
			"code": 403,
			"msg":  "无权转接当前会话",
		})
		return
	}
	previousOwnerID := visitor.ToId
	assignResult := routing.GetDefaultCenter().TransferSession(visitorId, kefuId)
	if !assignResult.Assigned {
		assignResult = routing.GetDefaultCenter().AssignSession(routing.AssignmentRequest{
			VisitorID:        visitorId,
			VisitorName:      visitor.Name,
			PreferredOwnerID: kefuId,
		})
	}
	if !assignResult.Assigned {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  assignResult.Reason,
		})
		return
	}
	models.UpdateVisitorKefu(visitorId, kefuId)
	visitor.ToId = kefuId
	ws.UpdateVisitorUser(visitorId, kefuId)
	go ws.VisitorOnline(kefuId, visitor)
	if previousOwnerID != "" && previousOwnerID != kefuId {
		go ws.NotifyKefuVisitorOffline(previousOwnerID, visitor.VisitorId, visitor.Name)
	}
	if sessionSnapshot, exists := routing.GetDefaultCenter().GetSession(visitorId); exists {
		go ws.BroadcastSessionUpdated(sessionSnapshot)
	}
	RecordAuditLog(c, "session.transferred", "session", visitorId, gin.H{
		"previous_owner_id": previousOwnerID,
	}, gin.H{
		"owner_id": kefuId,
	})
	outbox.EnqueueSessionScopedEvent(outbox.EventSessionTransferred, "session", visitorId)
	go ws.VisitorNotice(visitor.VisitorId, "客服转接到"+user.Nickname)
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "转移成功",
	})
}

func PostTakeSession(c *gin.Context) {
	visitorId := c.PostForm("visitor_id")
	currentKefuID, _ := c.Get("kefu_name")
	if visitorId == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "visitor_id不能为空",
		})
		return
	}

	currentKefuName := currentKefuID.(string)
	currentKefu := models.FindUser(currentKefuName)
	visitor := models.FindVisitorByVistorId(visitorId)
	if currentKefu.Name == "" || visitor.Name == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "访客或客服不存在",
		})
		return
	}
	sessionSnapshot, hasSession := routing.GetDefaultCenter().GetSession(visitorId)
	routeStatus := ""
	ownerID := visitor.ToId
	if hasSession {
		routeStatus = sessionSnapshot.RouteStatus
		if sessionSnapshot.OwnerID != "" {
			ownerID = sessionSnapshot.OwnerID
		}
	}
	if !middleware.CanAccessSession(c, ownerID, routeStatus) {
		c.JSON(200, gin.H{
			"code": 403,
			"msg":  "无权接管当前会话",
		})
		return
	}

	previousOwnerID := visitor.ToId
	assignResult := routing.GetDefaultCenter().TransferSession(visitorId, currentKefuName)
	if !assignResult.Assigned {
		assignResult = routing.GetDefaultCenter().AssignSession(routing.AssignmentRequest{
			VisitorID:        visitorId,
			VisitorName:      visitor.Name,
			PreferredOwnerID: currentKefuName,
		})
	}
	if !assignResult.Assigned {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  assignResult.Reason,
		})
		return
	}

	models.UpdateVisitorKefu(visitorId, currentKefuName)
	visitor.ToId = currentKefuName
	ws.UpdateVisitorUser(visitorId, currentKefuName)
	go ws.VisitorOnline(currentKefuName, visitor)
	if previousOwnerID != "" && previousOwnerID != currentKefuName {
		go ws.NotifyKefuVisitorOffline(previousOwnerID, visitor.VisitorId, visitor.Name)
	}
	if sessionSnapshot, exists := routing.GetDefaultCenter().GetSession(visitorId); exists {
		go ws.BroadcastSessionUpdated(sessionSnapshot)
	}
	RecordAuditLog(c, "session.taken", "session", visitorId, gin.H{
		"previous_owner_id": previousOwnerID,
	}, gin.H{
		"owner_id": currentKefuName,
	})
	outbox.EnqueueSessionScopedEvent(outbox.EventSessionAssigned, "session", visitorId)
	go ws.VisitorNotice(visitor.VisitorId, "客服 "+currentKefu.Nickname+" 已接管当前会话")

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "接管成功",
		"result": gin.H{
			"owner_id": currentKefuName,
		},
	})
}
func GetKefuInfoSetting(c *gin.Context) {
	kefuId := c.Query("kefu_id")
	currentKefuIDValue, _ := c.Get("kefu_id")
	if !middleware.HasPermissionFromContext(c, middleware.PermissionUserManage) && kefuId != "" && kefuId != stringifyAuditValue(currentKefuIDValue) {
		c.JSON(200, gin.H{
			"code": 403,
			"msg":  "无权查看其他客服资料",
		})
		return
	}
	user := models.FindUserById(kefuId)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": user,
	})
}
func PostKefuRegister(c *gin.Context) {
	if !models.IsSetupReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":   http.StatusServiceUnavailable,
			"msg":    "系统未完成安装，请先访问 /install 初始化数据库",
			"result": nil,
		})
		return
	}

	name := c.PostForm("username")
	password := c.PostForm("password")
	nickname := c.PostForm("nickname")
	avatar := "/static/images/4.jpg"

	if name == "" || password == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":   400,
			"msg":    "All fields are required",
			"result": nil,
		})
		return
	}

	existingUser := models.FindUser(name)
	if existingUser.Name != "" {
		c.JSON(http.StatusOK, gin.H{
			"code":   409,
			"msg":    "Username already exists",
			"result": nil,
		})
		return
	}

	hashedPassword, hashError := tools.HashPassword(password)
	if hashError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":   500,
			"msg":    "Password hashing failed",
			"result": nil,
		})
		return
	}

	userID := models.CreateUser(name, hashedPassword, avatar, nickname)
	if userID == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":   500,
			"msg":    "Registration Failed, please verify the database schema is initialized",
			"result": nil,
		})
		return
	}
	models.AssignRoleToUser(userID, "agent")

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "Registration successful",
		"result": gin.H{
			"user_id": userID,
		},
	})
}
func PostKefuInfo(c *gin.Context) {
	name, _ := c.Get("kefu_name")
	password := c.PostForm("password")
	avator := c.PostForm("avator")
	nickname := c.PostForm("nickname")
	if password != "" {
		hashedPassword, hashError := tools.HashPassword(password)
		if hashError != nil {
			c.JSON(200, gin.H{
				"code": 500,
				"msg":  "密码处理失败",
			})
			return
		}
		password = hashedPassword
	}
	if name == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "客服账号不能为空",
		})
		return
	}
	beforeUser := models.FindUser(name.(string))
	models.UpdateUser(name.(string), password, avator, nickname)
	RecordAuditLog(c, "user.profile.updated", "user", name.(string), gin.H{
		"nickname": beforeUser.Nickname,
		"avator":   beforeUser.Avator,
	}, gin.H{
		"nickname": nickname,
		"avator":   avator,
	})

	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": "",
	})
}

func PostKefuRoutingStatus(c *gin.Context) {
	kefuNameValue, _ := c.Get("kefu_name")
	kefuName := kefuNameValue.(string)
	presenceStatus := c.DefaultPostForm("presence_status", routing.PresenceOnline)
	acceptingRaw := c.DefaultPostForm("accepting_sessions", "true")
	acceptingSessions, parseError := strconv.ParseBool(acceptingRaw)
	if parseError != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "accepting_sessions格式错误",
		})
		return
	}
	if presenceStatus != routing.PresenceOnline && presenceStatus != routing.PresenceAway && presenceStatus != routing.PresenceBusy {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "presence_status不支持",
		})
		return
	}

	models.UpdateConfig(kefuName, "KefuPresenceStatus", presenceStatus)
	models.UpdateConfig(kefuName, "KefuAcceptingSessions", strconv.FormatBool(acceptingSessions))
	runtimeKefu, exists := routing.GetDefaultCenter().UpdateKefuRoutingStatus(kefuName, presenceStatus, acceptingSessions)
	if exists {
		go ws.BroadcastKefuStatusUpdated(runtimeKefu)
	}
	RecordAuditLog(c, "kefu.routing_status.updated", "user", kefuName, nil, gin.H{
		"presence_status":    presenceStatus,
		"accepting_sessions": acceptingSessions,
	})

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": gin.H{
			"presence_status":    presenceStatus,
			"accepting_sessions": acceptingSessions,
		},
	})
}
func GetKefuList(c *gin.Context) {
	users := models.FindUsers()
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "获取成功",
		"result": users,
	})
}
func DeleteKefuInfo(c *gin.Context) {
	kefuId := c.Query("id")
	user := models.FindUserById(kefuId)
	models.DeleteUserById(kefuId)
	models.DeleteRoleByUserId(kefuId)
	RecordAuditLog(c, "user.deleted", "user", kefuId, gin.H{
		"name":     user.Name,
		"nickname": user.Nickname,
	}, nil)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "删除成功",
		"result": "",
	})
}
