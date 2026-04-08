package ws

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"goflylivechat/agent"
	"goflylivechat/common"
	"goflylivechat/models"
	"goflylivechat/tools"
	"log"
	"time"
)

func NewVisitorServer(c *gin.Context) {
	//go kefuServerBackend()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	//获取GET参数,创建WS
	vistorInfo := models.FindVisitorByVistorId(c.Query("visitor_id"))
	if vistorInfo.VisitorId == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "访客不存在",
		})
		return
	}
	user := &User{
		Conn:       conn,
		Name:       vistorInfo.Name,
		Avator:     vistorInfo.Avator,
		Id:         vistorInfo.VisitorId,
		To_id:      vistorInfo.ToId,
		UpdateTime: time.Now(),
	}
	go models.UpdateVisitorStatus(vistorInfo.VisitorId, 1)
	//go SendServerJiang(vistorInfo.Name, "来了", c.Request.Host)

	AddVisitorToList(user)

	for {
		//接受消息
		var receive []byte
		messageType, receive, err := conn.ReadMessage()
		if err != nil {
			for _, visitor := range ClientList {
				if visitor.Conn == conn {
					log.Println("删除用户", visitor.Id)
					delete(ClientList, visitor.Id)
					VisitorOffline(visitor.To_id, visitor.Id, visitor.Name)
				}
			}
			log.Println(err)
			return
		}

		message <- &Message{
			conn:        conn,
			content:     receive,
			context:     c,
			messageType: messageType,
		}
	}
}
func AddVisitorToList(user *User) {
	//用户id对应的连接
	oldUser, ok := ClientList[user.Id]
	if oldUser != nil || ok {
		msg := TypeMessage{
			Type: "close",
			Data: user.Id,
		}
		str, _ := json.Marshal(msg)
		if err := oldUser.Conn.WriteMessage(websocket.TextMessage, str); err != nil {
			oldUser.Conn.Close()
			user.UpdateTime = oldUser.UpdateTime
			delete(ClientList, user.Id)
		}
	}
	ClientList[user.Id] = user
	lastMessage := models.FindLastMessageByVisitorId(user.Id)
	userInfo := make(map[string]string)
	userInfo["uid"] = user.Id
	userInfo["username"] = user.Name
	userInfo["avator"] = user.Avator
	userInfo["last_message"] = lastMessage.Content
	if userInfo["last_message"] == "" {
		userInfo["last_message"] = "new visitor"
	}
	msg := TypeMessage{
		Type: "userOnline",
		Data: userInfo,
	}
	str, _ := json.Marshal(msg)

	//新版
	OneKefuMessage(user.To_id, str)
}
func VisitorOnline(kefuId string, visitor models.Visitor) {
	lastMessage := models.FindLastMessageByVisitorId(visitor.VisitorId)
	userInfo := make(map[string]string)
	userInfo["uid"] = visitor.VisitorId
	userInfo["username"] = visitor.Name
	userInfo["avator"] = visitor.Avator
	userInfo["last_message"] = lastMessage.Content
	if userInfo["last_message"] == "" {
		userInfo["last_message"] = "new visitor"
	}
	msg := TypeMessage{
		Type: "userOnline",
		Data: userInfo,
	}
	str, _ := json.Marshal(msg)
	OneKefuMessage(kefuId, str)
}
func VisitorOffline(kefuId string, visitorId string, visitorName string) {

	models.UpdateVisitorStatus(visitorId, 0)
	releaseAgentSession(visitorId)
	userInfo := make(map[string]string)
	userInfo["uid"] = visitorId
	userInfo["name"] = visitorName
	msg := TypeMessage{
		Type: "userOffline",
		Data: userInfo,
	}
	str, _ := json.Marshal(msg)
	//新版
	OneKefuMessage(kefuId, str)
}
func VisitorNotice(visitorId string, notice string) {
	msg := TypeMessage{
		Type: "notice",
		Data: notice,
	}
	str, _ := json.Marshal(msg)
	visitor, ok := ClientList[visitorId]
	if !ok || visitor == nil || visitor.Conn == nil {
		return
	}
	visitor.Conn.WriteMessage(websocket.TextMessage, str)
}
func VisitorMessage(visitorId, content string, kefuInfo models.User) {
	msg := TypeMessage{
		Type: "message",
		Data: ClientMessage{
			Name:    kefuInfo.Nickname,
			Avator:  kefuInfo.Avator,
			Id:      kefuInfo.Name,
			Time:    time.Now().Format("2006-01-02 15:04:05"),
			ToId:    visitorId,
			Content: content,
			IsKefu:  "yes",
		},
	}
	str, _ := json.Marshal(msg)
	visitor, ok := ClientList[visitorId]
	if !ok || visitor == nil || visitor.Conn == nil {
		return
	}
	visitor.Conn.WriteMessage(websocket.TextMessage, str)
}

// NotifyVisitorAgentAssigned 输入访客标识和 agent 展示名，输出为通知发送结果，目的在于在 agent 接管成功后向访客推送待命提醒。
func NotifyVisitorAgentAssigned(visitorID string, displayName string) {
	if visitorID == "" || displayName == "" {
		return
	}
	VisitorNotice(visitorID, "智能客服 "+displayName+" 已进入待命，将接续当前会话。")
}

// VisitorAutoReply 输入访客信息、客服信息和访客消息，输出为自动回复发送结果，目的在于命中 Redis 缓存的关键词规则后自动回复。
func VisitorAutoReply(vistorInfo models.Visitor, kefuInfo models.User, content string) {
	kefu, ok := KefuList[kefuInfo.Name]
	replyContent := MatchAutoReplyContent(kefuInfo.Name, content)
	if replyContent != "" {
		time.Sleep(1 * time.Second)
		VisitorMessage(vistorInfo.VisitorId, replyContent, kefuInfo)
		KefuMessage(vistorInfo.VisitorId, replyContent, kefuInfo)
		models.CreateMessage(kefuInfo.Name, vistorInfo.VisitorId, replyContent, "kefu")
	}
	if !ok || kefu == nil {
		time.Sleep(1 * time.Second)
		config := models.FindConfigByUserId(kefuInfo.Name, "OfflineMessage")
		if config.ConfValue != "" && replyContent == "" {
			VisitorMessage(vistorInfo.VisitorId, config.ConfValue, kefuInfo)
			models.CreateMessage(kefuInfo.Name, vistorInfo.VisitorId, config.ConfValue, "kefu")
			return
		}
		assignVisitorToAgent(vistorInfo)
	}
}
func CleanVisitorExpire() {
	go func() {
		log.Println("cleanVisitorExpire start...")
		for {
			for _, user := range ClientList {
				diff := time.Now().Sub(user.UpdateTime).Seconds()
				if diff >= common.VisitorExpire {
					msg := TypeMessage{
						Type: "auto_close",
						Data: user.Id,
					}
					str, _ := json.Marshal(msg)
					if err := user.Conn.WriteMessage(websocket.TextMessage, str); err != nil {
						user.Conn.Close()
						delete(ClientList, user.Id)
					}
					log.Println(user.Name + ":cleanVisitorExpire finshed")
				}
			}
			t := time.NewTimer(time.Second * 5)
			<-t.C
		}
	}()
}

// assignVisitorToAgent 输入访客信息，输出为 agent 分配尝试结果，目的在于为无人值守场景预留智能客服接待能力。
func assignVisitorToAgent(visitorInfo models.Visitor) {
	dispatcher := agent.GetDefaultDispatcher()
	if dispatcher == nil {
		return
	}
	assignError := dispatcher.AssignSession(tools.Ctx, agent.AssignRequest{
		VisitorID:   visitorInfo.VisitorId,
		VisitorName: visitorInfo.Name,
		Capability:  "chat",
		Source:      "visitor_auto_reply",
	})
	if assignError != nil {
		log.Printf("投递 agent 分配请求失败: %v", assignError)
	}
}

// releaseAgentSession 输入访客标识，输出为 agent 会话释放结果，目的在于在访客离线或会话结束后归还智能客服容量。
func releaseAgentSession(visitorID string) {
	dispatcher := agent.GetDefaultDispatcher()
	if dispatcher == nil || visitorID == "" {
		return
	}
	if releaseError := dispatcher.ReleaseSession(tools.Ctx, visitorID); releaseError != nil {
		log.Printf("投递 agent 释放请求失败: %v", releaseError)
	}
}
