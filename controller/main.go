package controller

import (
	"fmt"
	"goflylivechat/common"
	"goflylivechat/models"
	"goflylivechat/tools"
	"goflylivechat/ws"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

func PostInstall(c *gin.Context) {
	if models.IsSetupReady() {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "系统已经安装过了",
		})
		return
	}
	server := c.PostForm("server")
	port := c.PostForm("port")
	database := c.PostForm("database")
	username := c.PostForm("username")
	password := c.PostForm("password")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, server, port, database)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		log.Println(err)
		tools.Logger().Println(err)
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "数据库连接失败:" + err.Error(),
		})
		return
	}
	defer db.Close()
	isExist, _ := tools.IsFileExist(common.Dir)
	if !isExist {
		os.Mkdir(common.Dir, os.ModePerm)
	}
	fileConfig := common.MysqlConf
	file, fileError := os.OpenFile(fileConfig, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if fileError != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "数据库配置写入失败:" + fileError.Error(),
		})
		return
	}
	defer file.Close()

	format := `{
	"Server":"%s",
	"Port":"%s",
	"Database":"%s",
	"Username":"%s",
	"Password":"%s"
}
`
	data := fmt.Sprintf(format, server, port, database, username, password)
	if _, writeError := file.WriteString(data); writeError != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "数据库配置写入失败:" + writeError.Error(),
		})
		return
	}
	if connectError := models.Connect(); connectError != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  "数据库连接失败:" + connectError.Error(),
		})
		return
	}
	if installError := install(server, port, database, username, password); installError != nil {
		c.JSON(200, gin.H{
			"code": 400,
			"msg":  installError.Error(),
		})
		return
	}
	installFile, installFileError := os.OpenFile("./install.lock", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if installFileError == nil {
		_, _ = installFile.WriteString("gofly live chat")
		_ = installFile.Close()
	}
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "安装成功",
	})
}

func install(server, port, database, username, password string) error {
	importer := tools.ImportSqlTool{
		SqlPath:  "import.sql",
		Username: username,
		Password: password,
		Server:   server,
		Port:     port,
		Database: database,
	}
	return importer.ImportSql()
}

func GetStatistics(c *gin.Context) {
	visitors := models.CountVisitors()
	message := models.CountMessage(nil, nil)
	session := len(ws.ClientList)
	kefuNum := 0
	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "ok",
		"result": gin.H{
			"visitors": visitors,
			"message":  message,
			"session":  session + kefuNum,
		},
	})
}
