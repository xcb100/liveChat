package controller

import (
	"github.com/gin-gonic/gin"
	"goflylivechat/models"
	"goflylivechat/ws"
	"log"
)

type ReplyForm struct {
	GroupName string `form:"group_name" binding:"required"`
}
type ReplyContentForm struct {
	GroupId  string `form:"group_id" binding:"required"`
	Content  string `form:"content" binding:"required"`
	ItemName string `form:"item_name" binding:"required"`
}

// GetReplys 输入请求上下文，输出为当前客服的快捷回复分组，目的在于提供工作台回复数据。
func GetReplys(c *gin.Context) {
	kefuId, _ := c.Get("kefu_name")
	log.Println(kefuId)
	res := models.FindReplyByUserId(kefuId)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": res,
	})
}

// GetAutoReplys 输入请求上下文，输出为指定客服的自动回复标题列表，目的在于给访客端提供候选回复。
func GetAutoReplys(c *gin.Context) {
	kefu_id := c.Query("kefu_id")
	res := models.FindReplyTitleByUserId(kefu_id)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": res,
	})
}

// PostReply 输入请求上下文，输出为新增分组结果，目的在于创建快捷回复分组并刷新缓存。
func PostReply(c *gin.Context) {
	var replyForm ReplyForm
	kefuId, _ := c.Get("kefu_name")
	err := c.Bind(&replyForm)
	if err != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "error:" + err.Error(),
		})
		return
	}
	models.CreateReplyGroup(replyForm.GroupName, kefuId.(string))
	ws.InvalidateAutoReplyCache(kefuId.(string))
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}

// PostReplyContent 输入请求上下文，输出为新增回复内容结果，目的在于创建关键词回复并刷新缓存。
func PostReplyContent(c *gin.Context) {
	var replyContentForm ReplyContentForm
	kefuId, _ := c.Get("kefu_name")
	err := c.Bind(&replyContentForm)
	if err != nil {
		c.JSON(400, gin.H{
			"code": 200,
			"msg":  "error:" + err.Error(),
		})
		return
	}
	models.CreateReplyContent(replyContentForm.GroupId, kefuId.(string), replyContentForm.Content, replyContentForm.ItemName)
	ws.InvalidateAutoReplyCache(kefuId.(string))
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}

// PostReplyContentSave 输入请求上下文，输出为更新回复内容结果，目的在于修改快捷回复并刷新缓存。
func PostReplyContentSave(c *gin.Context) {
	kefuId, _ := c.Get("kefu_name")
	replyId := c.PostForm("reply_id")
	replyTitle := c.PostForm("reply_title")
	replyContent := c.PostForm("reply_content")
	if replyId == "" || replyTitle == "" || replyContent == "" {
		c.JSON(400, gin.H{
			"code": 200,
			"msg":  "参数错误!",
		})
		return
	}
	models.UpdateReplyContent(replyId, kefuId.(string), replyTitle, replyContent)
	ws.InvalidateAutoReplyCache(kefuId.(string))
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}

// DelReplyContent 输入请求上下文，输出为删除回复结果，目的在于删除单条快捷回复并刷新缓存。
func DelReplyContent(c *gin.Context) {
	kefuId, _ := c.Get("kefu_name")
	id := c.Query("id")
	models.DeleteReplyContent(id, kefuId.(string))
	ws.InvalidateAutoReplyCache(kefuId.(string))
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}

// DelReplyGroup 输入请求上下文，输出为删除分组结果，目的在于删除快捷回复分组并刷新缓存。
func DelReplyGroup(c *gin.Context) {
	kefuId, _ := c.Get("kefu_name")
	id := c.Query("id")
	models.DeleteReplyGroup(id, kefuId.(string))
	ws.InvalidateAutoReplyCache(kefuId.(string))
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
	})
}

// PostReplySearch 输入请求上下文，输出为搜索结果，目的在于按关键词和内容检索快捷回复。
func PostReplySearch(c *gin.Context) {
	kefuId, _ := c.Get("kefu_name")
	search := c.PostForm("search")
	if search == "" {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "参数错误",
		})
		return
	}
	res := models.FindReplyBySearch(kefuId, search)
	c.JSON(200, gin.H{
		"code":   200,
		"msg":    "ok",
		"result": res,
	})
}
