package router

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"goflylivechat/models"
)

const spaEntryPath = "./static/dist/index.html"

// InitViewRouter 输入 Gin 引擎，输出为页面路由注册结果，目的在于将后台、安装页和访客页统一收敛到 Vue SPA 入口。
func InitViewRouter(engine *gin.Engine) {
	engine.GET("/", PageLogin)
	engine.GET("/login", PageLogin)
	engine.GET("/install", PageInstall)
	engine.GET("/pannel", PagePannel)
	engine.GET("/livechat", PageChat)
	engine.GET("/main", PageMain)
	engine.GET("/chat_main", PageChatMain)
	engine.GET("/setting", PageSetting)
}

// serveSPAIndex 输入请求上下文，输出为 Vue 入口文件内容，目的在于在显式页面路由上返回统一的前端构建产物。
func serveSPAIndex(c *gin.Context) {
	if _, statError := os.Stat(spaEntryPath); statError != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code": http.StatusServiceUnavailable,
			"msg":  "前端构建产物不存在，请先执行 npm run build",
		})
		return
	}

	c.File(spaEntryPath)
}

// PageLogin 输入请求上下文，输出为登录页 SPA 入口，目的在于提供后台登录入口。
func PageLogin(c *gin.Context) {
	if !models.IsSetupReady() {
		c.Redirect(http.StatusTemporaryRedirect, "/install")
		return
	}
	serveSPAIndex(c)
}

// PageInstall 输入请求上下文，输出为安装页 SPA 入口，目的在于提供数据库初始化入口。
func PageInstall(c *gin.Context) {
	if models.IsSetupReady() {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	serveSPAIndex(c)
}

// PagePannel 输入请求上下文，输出为统计页重定向结果，目的在于将旧看板页收敛到新的工作台入口。
func PagePannel(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/main")
}

// PageMain 输入请求上下文，输出为工作台 SPA 入口，目的在于展示新的客服主界面。
func PageMain(c *gin.Context) {
	if !models.IsSetupReady() {
		c.Redirect(http.StatusTemporaryRedirect, "/install")
		return
	}
	serveSPAIndex(c)
}

// PageChat 输入请求上下文，输出为访客聊天页 SPA 入口，目的在于保留访客端入口。
func PageChat(c *gin.Context) {
	if !models.IsSetupReady() {
		c.Redirect(http.StatusTemporaryRedirect, "/install")
		return
	}
	serveSPAIndex(c)
}

// PageChatMain 输入请求上下文，输出为重定向结果，目的在于兼容旧客服台地址。
func PageChatMain(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/main")
}

// PageSetting 输入请求上下文，输出为重定向结果，目的在于将旧设置页收敛到新的资料面板。
func PageSetting(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/main?panel=profile")
}
