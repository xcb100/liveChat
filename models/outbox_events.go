package models

import (
	"time"

	"github.com/jinzhu/gorm"
)

const (
	OutboxStatusPending    = "pending"
	OutboxStatusProcessing = "processing"
	OutboxStatusPublished  = "published"
	OutboxStatusFailed     = "failed"
)

type OutboxEvent struct {
	ID            uint       `gorm:"primary_key" json:"id"`
	EventType     string     `json:"event_type"`
	AggregateType string     `json:"aggregate_type"`
	AggregateID   string     `json:"aggregate_id"`
	Payload       string     `gorm:"type:text" json:"payload"`
	Status        string     `json:"status"`
	Attempts      uint       `json:"attempts"`
	LastError     string     `gorm:"type:text" json:"last_error"`
	NextRetryAt   *time.Time `json:"next_retry_at"`
	PublishedAt   *time.Time `json:"published_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func CreateOutboxEvent(outboxEvent *OutboxEvent) uint {
	if outboxEvent == nil || !IsDatabaseReachable() {
		return 0
	}
	if outboxEvent.Status == "" {
		outboxEvent.Status = OutboxStatusPending
	}
	if outboxEvent.CreatedAt.IsZero() {
		outboxEvent.CreatedAt = time.Now()
	}
	outboxEvent.UpdatedAt = time.Now()
	DB.Create(outboxEvent)
	return outboxEvent.ID
}

func FindPendingOutboxEvents(limit int, readyAt time.Time, maxAttempts uint) []OutboxEvent {
	outboxEvents := make([]OutboxEvent, 0)
	if !IsDatabaseReachable() {
		return outboxEvents
	}
	if limit <= 0 {
		limit = 20
	}
	DB.Where(
		"(status = ? OR status = ?) AND attempts < ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
		OutboxStatusPending,
		OutboxStatusFailed,
		maxAttempts,
		readyAt,
	).Order("id asc").Limit(limit).Find(&outboxEvents)
	return outboxEvents
}

func MarkOutboxEventProcessing(id uint) bool {
	if !IsDatabaseReachable() || id == 0 {
		return false
	}
	return DB.Model(&OutboxEvent{}).
		Where("id = ? AND status IN (?)", id, []string{OutboxStatusPending, OutboxStatusFailed}).
		Updates(map[string]interface{}{
			"status":     OutboxStatusProcessing,
			"updated_at": time.Now(),
		}).RowsAffected > 0
}

func MarkOutboxEventPublished(id uint) bool {
	if !IsDatabaseReachable() || id == 0 {
		return false
	}
	now := time.Now()
	return DB.Model(&OutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       OutboxStatusPublished,
		"published_at": now,
		"last_error":   "",
		"updated_at":   now,
	}).RowsAffected > 0
}

func MarkOutboxEventFailed(id uint, attempts uint, lastError string, nextRetryAt time.Time) bool {
	if !IsDatabaseReachable() || id == 0 {
		return false
	}
	return DB.Model(&OutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        OutboxStatusFailed,
		"attempts":      attempts,
		"last_error":    lastError,
		"next_retry_at": nextRetryAt,
		"updated_at":    time.Now(),
	}).RowsAffected > 0
}

func IncrementOutboxEventAttempts(id uint) bool {
	if !IsDatabaseReachable() || id == 0 {
		return false
	}
	return DB.Model(&OutboxEvent{}).Where("id = ?", id).UpdateColumn("attempts", gorm.Expr("attempts + ?", 1)).RowsAffected > 0
}

func FindOutboxEvents(page uint, pagesize uint) []OutboxEvent {
	if page == 0 {
		page = 1
	}
	if pagesize == 0 {
		pagesize = 20
	}
	offset := (page - 1) * pagesize
	outboxEvents := make([]OutboxEvent, 0)
	if !IsDatabaseReachable() {
		return outboxEvents
	}
	DB.Order("id desc").Offset(offset).Limit(pagesize).Find(&outboxEvents)
	return outboxEvents
}

func CountOutboxEvents() uint {
	var count uint
	if !IsDatabaseReachable() {
		return count
	}
	DB.Model(&OutboxEvent{}).Count(&count)
	return count
}
