package ws

import (
	"encoding/json"
	"goflylivechat/models"
	"goflylivechat/routing"
	"goflylivechat/tools"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func NewKefuServer(c *gin.Context) {
	kefuName, _ := c.Get("kefu_name")
	kefuInfo := models.FindUser(kefuName.(string))
	if kefuInfo.ID == 0 {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "用户不存在",
		})
		return
	}

	//go kefuServerBackend()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	//获取GET参数,创建WS
	var kefu User
	kefu.Id = kefuInfo.Name
	kefu.Name = kefuInfo.Nickname
	kefu.Avator = kefuInfo.Avator
	kefu.Conn = conn
	AddKefuToList(&kefu)

	for {
		//接受消息
		var receive []byte
		messageType, receive, err := conn.ReadMessage()
		if err != nil {
			log.Println("ws/user.go ", err)
			conn.Close()
			RemoveKefuFromList(kefu.Id)
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
func AddKefuToList(kefu *User) {
	oldUser, ok := KefuList[kefu.Id]
	if oldUser != nil || ok {
		msg := TypeMessage{
			Type: "close",
			Data: kefu.Id,
		}
		str, _ := json.Marshal(msg)
		if err := oldUser.Conn.WriteMessage(websocket.TextMessage, str); err != nil {
			oldUser.Conn.Close()
		}
	}
	KefuList[kefu.Id] = kefu
	routing.GetDefaultCenter().MarkKefuOnline(kefu.Id, kefu.Name)
}

func RemoveKefuFromList(kefuID string) {
	delete(KefuList, kefuID)
	routing.GetDefaultCenter().MarkKefuOffline(kefuID)
}

// 给指定客服发消息
func OneKefuMessage(toId string, str []byte) {
	kefu, ok := KefuList[toId]
	if ok {
		log.Println("OneKefuMessage lock")
		kefu.Mux.Lock()
		defer func() {
			kefu.Mux.Unlock()
			log.Println("OneKefuMessage unlock")
		}()
		error := kefu.Conn.WriteMessage(websocket.TextMessage, str)
		tools.Logger().Println("send_kefu_message", error, string(str))
	}
}

// BroadcastKefuMessage 输入消息体，输出为广播结果，目的在于把会话路由变更同步给所有在线客服工作台。
func BroadcastKefuMessage(str []byte) {
	for kefuID := range KefuList {
		OneKefuMessage(kefuID, str)
	}
}

// BroadcastKefuStatusUpdated 输入客服运行时状态，输出为广播结果，目的在于把坐席在线/接待状态变化同步给所有工作台。
func BroadcastKefuStatusUpdated(runtimeKefu routing.RuntimeKefu) {
	msg := TypeMessage{
		Type: "kefuStatusUpdated",
		Data: map[string]interface{}{
			"kefu_id":            runtimeKefu.KefuID,
			"display_name":       runtimeKefu.DisplayName,
			"skills":             runtimeKefu.Skills,
			"presence_status":    runtimeKefu.PresenceStatus,
			"accepting_sessions": runtimeKefu.AcceptingSessions,
			"active_sessions":    runtimeKefu.ActiveSessions,
			"max_sessions":       runtimeKefu.MaxSessions,
		},
	}
	str, _ := json.Marshal(msg)
	BroadcastKefuMessage(str)
}

func KefuMessage(visitorId, content string, kefuInfo models.User) {
	msg := TypeMessage{
		Type: "message",
		Data: ClientMessage{
			Name:    kefuInfo.Nickname,
			Avator:  kefuInfo.Avator,
			Id:      visitorId,
			Time:    time.Now().Format("2006-01-02 15:04:05"),
			ToId:    visitorId,
			Content: content,
			IsKefu:  "yes",
		},
	}
	str, _ := json.Marshal(msg)
	OneKefuMessage(kefuInfo.Name, str)
}

// 给客服客户端发送消息判断客户端是否在线
func SendPingToKefuClient() {
	msg := TypeMessage{
		Type: "many pong",
	}
	str, _ := json.Marshal(msg)
	for kefuId, kefu := range KefuList {
		if kefu == nil {
			continue
		}
		kefu.Mux.Lock()
		defer kefu.Mux.Unlock()
		err := kefu.Conn.WriteMessage(websocket.TextMessage, str)
		if err != nil {
			log.Println("定时发送ping给客服，失败", err.Error())
			RemoveKefuFromList(kefuId)
		}
	}
}
