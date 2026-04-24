package models

import (
	"context"
	"fmt"
	"goflylivechat/common"
	"goflylivechat/tools"
	"log"
	"time"

	"github.com/jinzhu/gorm"
	"go.opentelemetry.io/otel/attribute"
)

var DB *gorm.DB

var requiredTables = []string{
	"user",
	"visitor",
	"message",
	"config",
	"reply_group",
	"reply_item",
	"ipblack",
	"role",
	"user_role",
	"audit_log",
	"conversation_session",
	"session_summary",
	"outbox_event",
}

type Model struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"deleted_at"`
}

func init() {
	if connectError := Connect(); connectError != nil {
		log.Printf("数据库初始化未完成: %v", connectError)
	}
}

// Connect 输入为空，输出为数据库连接结果，目的在于按当前配置初始化全局 Gorm 连接。
func Connect() error {
	mysql := common.GetMysqlConf()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", mysql.Username, mysql.Password, mysql.Server, mysql.Port, mysql.Database)
	var err error
	DB, err = gorm.Open("mysql", dsn)
	if err != nil {
		log.Println(err)
		return err
	}
	DB.SingularTable(true)
	DB.LogMode(true)
	DB.DB().SetMaxIdleConns(10)
	DB.DB().SetMaxOpenConns(100)
	DB.DB().SetConnMaxLifetime(59 * time.Second)
	return nil
}

// IsDatabaseReachable 输入为空，输出为数据库连接可用性，目的在于统一判断当前 Gorm 连接是否仍然可用。
func IsDatabaseReachable() bool {
	if DB == nil {
		return false
	}
	sqlDB := DB.DB()
	if sqlDB == nil {
		return false
	}
	return sqlDB.Ping() == nil
}

// HasRequiredTables 输入为空，输出为核心业务表是否完整，目的在于避免仅凭 install.lock 误判系统已完成安装。
func HasRequiredTables() bool {
	if !IsDatabaseReachable() {
		return false
	}
	for _, tableName := range requiredTables {
		if !DB.HasTable(tableName) {
			return false
		}
	}
	return true
}

// IsSetupReady 输入为空，输出为系统安装是否完成，目的在于统一给启动、页面路由和接口做安装态判断。
func IsSetupReady() bool {
	return HasRequiredTables()
}

// Execute 输入上下文和 SQL 语句，输出为执行结果，目的在于统一封装带 tracing 的 SQL 执行入口。
func Execute(ctx context.Context, sql string) error {
	ctx, span := tools.Tracer.Start(ctx, "DB.Execute")
	span.SetAttributes(
		attribute.String("sql", sql),
	)
	defer span.End()
	return DB.Exec(sql).Error
}

// CloseDB 输入为空，输出为数据库关闭结果，目的在于在测试或进程退出时释放数据库连接。
func CloseDB() {
	if DB == nil {
		return
	}
	defer DB.Close()
}
