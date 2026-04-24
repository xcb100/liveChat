package models

import "time"

type SessionSummary struct {
	ID                  uint       `gorm:"primary_key" json:"id"`
	SessionID           string     `json:"session_id"`
	VisitorID           string     `json:"visitor_id"`
	DisplayName         string     `json:"display_name"`
	Avatar              string     `json:"avatar"`
	OwnerID             string     `json:"owner_id"`
	StickyOwnerID       string     `json:"sticky_owner_id"`
	RouteStatus         string     `json:"route_status"`
	QueueName           string     `json:"queue_name"`
	PreferredSkill      string     `json:"preferred_skill"`
	LastMessage         string     `json:"last_message"`
	LastMessageAt       *time.Time `json:"last_message_at"`
	UnreadCount         uint       `json:"unread_count"`
	LastRouteReason     string     `json:"last_route_reason"`
	VisitorStatus       uint       `json:"visitor_status"`
	QueueEnteredAt      *time.Time `json:"queue_entered_at"`
	LastAssignAttemptAt *time.Time `json:"last_assign_attempt_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	CreatedAt           time.Time  `json:"created_at"`
}

func SaveSessionSummary(sessionSummary *SessionSummary) uint {
	if sessionSummary == nil || !IsDatabaseReachable() {
		return 0
	}

	existingSessionSummary := SessionSummary{}
	DB.Where("session_id = ?", sessionSummary.SessionID).First(&existingSessionSummary)
	if existingSessionSummary.ID != 0 {
		sessionSummary.ID = existingSessionSummary.ID
		sessionSummary.CreatedAt = existingSessionSummary.CreatedAt
	} else if sessionSummary.CreatedAt.IsZero() {
		sessionSummary.CreatedAt = time.Now()
	}
	DB.Save(sessionSummary)
	return sessionSummary.ID
}

func FindSessionSummariesByFilter(ownerID string, routeStatus string) []SessionSummary {
	sessionSummaries := make([]SessionSummary, 0)
	if !IsDatabaseReachable() {
		return sessionSummaries
	}
	dbQuery := DB.Order("updated_at desc")
	if ownerID != "" {
		dbQuery = dbQuery.Where("owner_id = ?", ownerID)
	}
	if routeStatus != "" {
		dbQuery = dbQuery.Where("route_status = ?", routeStatus)
	}
	dbQuery.Find(&sessionSummaries)
	return sessionSummaries
}

func FindSessionSummaryByVisitorID(visitorID string) SessionSummary {
	sessionSummary := SessionSummary{}
	if !IsDatabaseReachable() {
		return sessionSummary
	}
	DB.Where("visitor_id = ?", visitorID).Order("updated_at desc").First(&sessionSummary)
	return sessionSummary
}

func SyncSessionSummaryByVisitorID(visitorID string) {
	if !IsDatabaseReachable() || visitorID == "" {
		return
	}

	conversationSession := FindLatestConversationSessionByVisitorID(visitorID)
	if conversationSession.ID == 0 {
		return
	}

	visitor := FindVisitorByVistorId(visitorID)
	lastMessage := FindLastMessageByVisitorId(visitorID)
	displayName := conversationSession.VisitorName
	if displayName == "" {
		displayName = visitor.Name
	}
	if displayName == "" {
		displayName = "访客"
	}
	avatar := visitor.Avator
	if avatar == "" {
		avatar = "/static/images/2.png"
	}
	lastMessageContent := lastMessage.Content
	if lastMessageContent == "" {
		lastMessageContent = visitor.LastMessage
	}
	if lastMessageContent == "" {
		lastMessageContent = "new visitor"
	}
	lastMessageAt := timePointer(lastMessage.CreatedAt)
	if lastMessageAt == nil {
		lastMessageAt = timePointer(conversationSession.LastActivityAtValue())
	}

	SaveSessionSummary(&SessionSummary{
		SessionID:           conversationSession.SessionID,
		VisitorID:           conversationSession.VisitorID,
		DisplayName:         displayName,
		Avatar:              avatar,
		OwnerID:             conversationSession.OwnerID,
		StickyOwnerID:       conversationSession.StickyOwnerID,
		RouteStatus:         conversationSession.RouteStatus,
		QueueName:           conversationSession.QueueName,
		PreferredSkill:      conversationSession.PreferredSkill,
		LastMessage:         lastMessageContent,
		LastMessageAt:       lastMessageAt,
		UnreadCount:         FindUnreadMessageNumByVisitorId(visitorID),
		LastRouteReason:     conversationSession.LastRouteReason,
		VisitorStatus:       visitor.Status,
		QueueEnteredAt:      conversationSession.QueueEnteredAt,
		LastAssignAttemptAt: conversationSession.LastAssignAttemptAt,
		UpdatedAt:           time.Now(),
	})
}

func timePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	valueCopy := value
	return &valueCopy
}
