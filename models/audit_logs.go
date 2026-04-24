package models

import "time"

type AuditLog struct {
	ID          uint      `gorm:"primary_key" json:"id"`
	ActorUserID uint      `json:"actor_user_id"`
	ActorName   string    `json:"actor_name"`
	ActorRole   string    `json:"actor_role"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	BeforeData  string    `gorm:"type:text" json:"before_data"`
	AfterData   string    `gorm:"type:text" json:"after_data"`
	ClientIP    string    `json:"client_ip"`
	RequestID   string    `json:"request_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func CreateAuditLog(auditLog *AuditLog) uint {
	if auditLog == nil {
		return 0
	}
	if auditLog.CreatedAt.IsZero() {
		auditLog.CreatedAt = time.Now()
	}
	DB.Create(auditLog)
	return auditLog.ID
}

func FindAuditLogs(page uint, pagesize uint) []AuditLog {
	offset := (page - 1) * pagesize
	if offset < 0 {
		offset = 0
	}
	var auditLogs []AuditLog
	DB.Order("id desc").Offset(offset).Limit(pagesize).Find(&auditLogs)
	return auditLogs
}

func CountAuditLogs() uint {
	var count uint
	DB.Model(&AuditLog{}).Count(&count)
	return count
}
