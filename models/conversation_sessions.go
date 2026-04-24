package models

import "time"

type ConversationSession struct {
	ID                  uint       `gorm:"primary_key" json:"id"`
	SessionID           string     `json:"session_id"`
	VisitorID           string     `json:"visitor_id"`
	VisitorName         string     `json:"visitor_name"`
	OwnerID             string     `json:"owner_id"`
	StickyOwnerID       string     `json:"sticky_owner_id"`
	RouteStatus         string     `json:"route_status"`
	QueueName           string     `json:"queue_name"`
	PreferredSkill      string     `json:"preferred_skill"`
	SourceEntry         string     `json:"source_entry"`
	ServedByType        string     `json:"served_by_type"`
	LastRouteReason     string     `json:"last_route_reason"`
	QueueEnteredAt      *time.Time `json:"queue_entered_at"`
	LastAssignAttemptAt *time.Time `json:"last_assign_attempt_at"`
	LastAssignedAt      *time.Time `json:"last_assigned_at"`
	LastTransferAt      *time.Time `json:"last_transfer_at"`
	LastActivityAt      *time.Time `json:"last_activity_at"`
	ClosedAt            *time.Time `json:"closed_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func SaveConversationSession(conversationSession *ConversationSession) uint {
	if conversationSession == nil || !IsDatabaseReachable() {
		return 0
	}

	existingConversationSession := ConversationSession{}
	DB.Where("session_id = ?", conversationSession.SessionID).First(&existingConversationSession)
	if existingConversationSession.ID != 0 {
		conversationSession.ID = existingConversationSession.ID
		conversationSession.CreatedAt = existingConversationSession.CreatedAt
	} else if conversationSession.CreatedAt.IsZero() {
		conversationSession.CreatedAt = time.Now()
	}
	DB.Save(conversationSession)
	return conversationSession.ID
}

func FindOpenConversationSessions() []ConversationSession {
	conversationSessions := make([]ConversationSession, 0)
	if !IsDatabaseReachable() {
		return conversationSessions
	}
	DB.Where("route_status <> ?", "closed").Order("updated_at asc").Find(&conversationSessions)
	return conversationSessions
}

func FindLatestConversationSessionByVisitorID(visitorID string) ConversationSession {
	conversationSession := ConversationSession{}
	if !IsDatabaseReachable() {
		return conversationSession
	}
	DB.Where("visitor_id = ?", visitorID).Order("updated_at desc").First(&conversationSession)
	return conversationSession
}

func (conversationSession ConversationSession) LastActivityAtValue() time.Time {
	if conversationSession.LastActivityAt == nil {
		return time.Time{}
	}
	return *conversationSession.LastActivityAt
}
