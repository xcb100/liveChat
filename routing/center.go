package routing

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"goflylivechat/models"
)

const (
	RouteStatusPending  = "pending"
	RouteStatusAssigned = "assigned"
	RouteStatusClosed   = "closed"

	PresenceOnline  = "online"
	PresenceAway    = "away"
	PresenceBusy    = "busy"
	PresenceOffline = "offline"

	ServedByHuman = "human"
)

type AssignmentRequest struct {
	VisitorID        string
	VisitorName      string
	PreferredOwnerID string
	PreferredSkill   string
	SourceEntry      string
	RequireAvailable bool
	AllowStickyOwner bool
	DefaultQueueName string
	ServedByType     string
}

type AssignmentResult struct {
	Assigned    bool   `json:"assigned"`
	OwnerID     string `json:"owner_id"`
	DisplayName string `json:"display_name"`
	RouteStatus string `json:"route_status"`
	QueueName   string `json:"queue_name"`
	Reason      string `json:"reason,omitempty"`
}

type SessionSnapshot struct {
	VisitorID           string    `json:"visitor_id"`
	VisitorName         string    `json:"visitor_name"`
	OwnerID             string    `json:"owner_id"`
	StickyOwnerID       string    `json:"sticky_owner_id"`
	RouteStatus         string    `json:"route_status"`
	QueueName           string    `json:"queue_name"`
	PreferredSkill      string    `json:"preferred_skill"`
	SourceEntry         string    `json:"source_entry"`
	ServedByType        string    `json:"served_by_type"`
	LastRouteReason     string    `json:"last_route_reason"`
	QueueEnteredAt      time.Time `json:"queue_entered_at"`
	LastAssignAttemptAt time.Time `json:"last_assign_attempt_at"`
	LastAssignedAt      time.Time `json:"last_assigned_at"`
	LastTransferAt      time.Time `json:"last_transfer_at"`
	LastActivityAt      time.Time `json:"last_activity_at"`
}

type SessionListFilter struct {
	OwnerID     string
	RouteStatus string
}

type RuntimeKefu struct {
	KefuID            string    `json:"kefu_id"`
	DisplayName       string    `json:"display_name"`
	Skills            []string  `json:"skills"`
	PresenceStatus    string    `json:"presence_status"`
	AcceptingSessions bool      `json:"accepting_sessions"`
	Enabled           bool      `json:"enabled"`
	MaxSessions       int       `json:"max_sessions"`
	ActiveSessions    int       `json:"active_sessions"`
	LastAssignedAt    time.Time `json:"last_assigned_at"`
	LastSeenAt        time.Time `json:"last_seen_at"`
}

type AutoDispatchConfig struct {
	RetryInterval time.Duration
	ExpandAfter   time.Duration
	PendingTTL    time.Duration
}

type Hooks struct {
	OnPendingAssigned func(SessionSnapshot)
}

type sessionState struct {
	VisitorID           string
	VisitorName         string
	OwnerID             string
	StickyOwnerID       string
	RouteStatus         string
	QueueName           string
	PreferredSkill      string
	SourceEntry         string
	ServedByType        string
	LastRouteReason     string
	QueueEnteredAt      time.Time
	LastAssignAttemptAt time.Time
	LastAssignedAt      time.Time
	LastTransferAt      time.Time
	LastActivityAt      time.Time
}

type Center struct {
	lock               sync.RWMutex
	sessions           map[string]*sessionState
	kefus              map[string]*RuntimeKefu
	defaultMaxSessions int
	defaultQueueName   string
	retryInterval      time.Duration
	expandAfter        time.Duration
	pendingTTL         time.Duration
	hooks              Hooks
}

var defaultCenter = NewCenter(5, "default", AutoDispatchConfig{})

func NewCenter(defaultMaxSessions int, defaultQueueName string, autoDispatchConfig AutoDispatchConfig) *Center {
	if defaultMaxSessions <= 0 {
		defaultMaxSessions = 5
	}
	if strings.TrimSpace(defaultQueueName) == "" {
		defaultQueueName = "default"
	}
	if autoDispatchConfig.RetryInterval <= 0 {
		autoDispatchConfig.RetryInterval = 3 * time.Second
	}
	if autoDispatchConfig.ExpandAfter <= 0 {
		autoDispatchConfig.ExpandAfter = 10 * time.Second
	}
	if autoDispatchConfig.PendingTTL <= 0 {
		autoDispatchConfig.PendingTTL = 10 * time.Minute
	}
	return &Center{
		sessions:           make(map[string]*sessionState),
		kefus:              make(map[string]*RuntimeKefu),
		defaultMaxSessions: defaultMaxSessions,
		defaultQueueName:   defaultQueueName,
		retryInterval:      autoDispatchConfig.RetryInterval,
		expandAfter:        autoDispatchConfig.ExpandAfter,
		pendingTTL:         autoDispatchConfig.PendingTTL,
	}
}

func ConfigureDefaultCenter(defaultMaxSessions int, defaultQueueName string, autoDispatchConfig AutoDispatchConfig) {
	defaultCenter = NewCenter(defaultMaxSessions, defaultQueueName, autoDispatchConfig)
}

func GetDefaultCenter() *Center {
	return defaultCenter
}

func (center *Center) SetHooks(hooks Hooks) {
	center.lock.Lock()
	defer center.lock.Unlock()
	center.hooks = hooks
}

func (center *Center) MarkKefuOnline(kefuID string, displayName string) RuntimeKefu {
	center.lock.Lock()

	runtimeKefu := center.ensureKefuLocked(kefuID, displayName)
	runtimeKefu.DisplayName = displayName
	runtimeKefu.Skills = models.GetUserRoutingSkills(kefuID)
	runtimeKefu.PresenceStatus = models.GetUserKefuPresenceStatus(kefuID)
	runtimeKefu.AcceptingSessions = models.GetUserKefuAcceptingSessions(kefuID)
	runtimeKefu.Enabled = true
	runtimeKefu.LastSeenAt = time.Now()
	runtimeKefu.ActiveSessions = center.activeSessionCountLocked(kefuID)
	_, assignedSnapshots := center.processPendingSessionsLocked(time.Now(), true)
	clonedRuntimeKefu := *runtimeKefu
	center.lock.Unlock()

	center.dispatchPendingAssigned(assignedSnapshots)
	return clonedRuntimeKefu
}

func (center *Center) UpdateKefuRoutingStatus(kefuID string, presenceStatus string, acceptingSessions bool) (RuntimeKefu, bool) {
	center.lock.Lock()
	defer center.lock.Unlock()

	runtimeKefu, exists := center.kefus[kefuID]
	if !exists {
		return RuntimeKefu{}, false
	}
	if !isValidPresenceStatus(presenceStatus) {
		presenceStatus = PresenceOnline
	}
	runtimeKefu.PresenceStatus = presenceStatus
	runtimeKefu.AcceptingSessions = acceptingSessions
	runtimeKefu.LastSeenAt = time.Now()
	runtimeKefu.ActiveSessions = center.activeSessionCountLocked(kefuID)
	clonedRuntimeKefu := *runtimeKefu
	clonedRuntimeKefu.Skills = append([]string(nil), runtimeKefu.Skills...)
	return clonedRuntimeKefu, true
}

func (center *Center) MarkKefuOffline(kefuID string) {
	center.lock.Lock()
	defer center.lock.Unlock()

	runtimeKefu, exists := center.kefus[kefuID]
	if !exists {
		return
	}
	runtimeKefu.PresenceStatus = PresenceOffline
	runtimeKefu.AcceptingSessions = false
	runtimeKefu.LastSeenAt = time.Now()
	runtimeKefu.ActiveSessions = center.activeSessionCountLocked(kefuID)
}

func (center *Center) AssignSession(request AssignmentRequest) AssignmentResult {
	center.lock.Lock()
	defer center.lock.Unlock()

	if strings.TrimSpace(request.VisitorID) == "" {
		return AssignmentResult{Assigned: false, RouteStatus: RouteStatusPending, QueueName: center.defaultQueueName, Reason: "visitor_id is required"}
	}

	session := center.ensureSessionLocked(request)
	now := time.Now()
	session.LastActivityAt = now
	if strings.TrimSpace(request.SourceEntry) != "" {
		session.SourceEntry = request.SourceEntry
	}
	if strings.TrimSpace(request.PreferredSkill) != "" {
		session.PreferredSkill = request.PreferredSkill
	}
	if strings.TrimSpace(request.ServedByType) != "" {
		session.ServedByType = request.ServedByType
	}
	if preferredOwnerID := strings.TrimSpace(request.PreferredOwnerID); preferredOwnerID != "" && (request.AllowStickyOwner || request.RequireAvailable) {
		session.StickyOwnerID = preferredOwnerID
	}

	if request.AllowStickyOwner && strings.TrimSpace(session.OwnerID) != "" && center.isKefuAvailableLocked(session.OwnerID) {
		return center.assignLocked(session, session.OwnerID, false)
	}

	preferredOwnerID := strings.TrimSpace(request.PreferredOwnerID)
	if preferredOwnerID != "" {
		if !request.RequireAvailable {
			return center.assignLocked(session, preferredOwnerID, true)
		}
		if center.isKefuAvailableLocked(preferredOwnerID) {
			return center.assignLocked(session, preferredOwnerID, true)
		}
	}

	selectedKefu := center.pickLeastLoadedKefuLocked(session.PreferredSkill)
	if selectedKefu == nil {
		reason := "暂无可用客服"
		if session.PreferredSkill != "" {
			reason = "当前技能池暂无可用客服"
		}
		return center.markPendingLocked(session, preferredOwnerID, reason)
	}

	return center.assignLocked(session, selectedKefu.KefuID, false)
}

func (center *Center) TransferSession(visitorID string, ownerID string) AssignmentResult {
	center.lock.Lock()
	defer center.lock.Unlock()

	session, exists := center.sessions[visitorID]
	if !exists {
		return AssignmentResult{Assigned: false, RouteStatus: RouteStatusPending, QueueName: center.defaultQueueName, Reason: "会话不存在"}
	}
	session.LastTransferAt = time.Now()
	return center.assignLocked(session, ownerID, true)
}

func (center *Center) ReleaseSession(visitorID string) bool {
	center.lock.Lock()
	defer center.lock.Unlock()

	session, exists := center.sessions[visitorID]
	if !exists {
		return false
	}
	session.RouteStatus = RouteStatusClosed
	session.LastRouteReason = ""
	session.LastActivityAt = time.Now()
	session.OwnerID = ""
	for _, runtimeKefu := range center.kefus {
		runtimeKefu.ActiveSessions = center.activeSessionCountLocked(runtimeKefu.KefuID)
	}
	return true
}

func (center *Center) GetSession(visitorID string) (SessionSnapshot, bool) {
	center.lock.RLock()
	defer center.lock.RUnlock()

	session, exists := center.sessions[visitorID]
	if !exists {
		return SessionSnapshot{}, false
	}
	return cloneSession(session), true
}

func (center *Center) ListKefus() []RuntimeKefu {
	center.lock.RLock()
	defer center.lock.RUnlock()

	runtimeKefus := make([]RuntimeKefu, 0, len(center.kefus))
	for _, runtimeKefu := range center.kefus {
		cloned := *runtimeKefu
		cloned.Skills = append([]string(nil), runtimeKefu.Skills...)
		cloned.ActiveSessions = center.activeSessionCountLocked(runtimeKefu.KefuID)
		runtimeKefus = append(runtimeKefus, cloned)
	}

	sort.SliceStable(runtimeKefus, func(leftIndex, rightIndex int) bool {
		if runtimeKefus[leftIndex].ActiveSessions == runtimeKefus[rightIndex].ActiveSessions {
			return runtimeKefus[leftIndex].LastAssignedAt.Before(runtimeKefus[rightIndex].LastAssignedAt)
		}
		return runtimeKefus[leftIndex].ActiveSessions < runtimeKefus[rightIndex].ActiveSessions
	})

	return runtimeKefus
}

func (center *Center) ListSessions(filter SessionListFilter) []SessionSnapshot {
	center.lock.RLock()
	defer center.lock.RUnlock()

	sessions := make([]SessionSnapshot, 0, len(center.sessions))
	for _, session := range center.sessions {
		if filter.OwnerID != "" && session.OwnerID != filter.OwnerID {
			continue
		}
		if filter.RouteStatus != "" && session.RouteStatus != filter.RouteStatus {
			continue
		}
		sessions = append(sessions, cloneSession(session))
	}

	sort.SliceStable(sessions, func(leftIndex, rightIndex int) bool {
		return sessions[leftIndex].LastActivityAt.After(sessions[rightIndex].LastActivityAt)
	})

	return sessions
}

func (center *Center) TouchSession(visitorID string) {
	center.lock.Lock()
	defer center.lock.Unlock()

	session, exists := center.sessions[visitorID]
	if !exists {
		return
	}
	session.LastActivityAt = time.Now()
}

func (center *Center) ProcessPendingSessions(now time.Time) int {
	center.lock.Lock()
	processedCount, assignedSnapshots := center.processPendingSessionsLocked(now, false)
	center.lock.Unlock()

	center.dispatchPendingAssigned(assignedSnapshots)
	return processedCount
}

func (center *Center) StartAutoDispatch(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(center.retryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case tickAt := <-ticker.C:
				center.ProcessPendingSessions(tickAt)
			}
		}
	}()
}

func (center *Center) ensureSessionLocked(request AssignmentRequest) *sessionState {
	session, exists := center.sessions[request.VisitorID]
	if exists {
		if strings.TrimSpace(request.VisitorName) != "" {
			session.VisitorName = request.VisitorName
		}
		return session
	}

	queueName := strings.TrimSpace(request.DefaultQueueName)
	if queueName == "" {
		queueName = center.defaultQueueName
	}
	servedByType := strings.TrimSpace(request.ServedByType)
	if servedByType == "" {
		servedByType = ServedByHuman
	}

	session = &sessionState{
		VisitorID:      request.VisitorID,
		VisitorName:    request.VisitorName,
		RouteStatus:    RouteStatusPending,
		QueueName:      queueName,
		PreferredSkill: request.PreferredSkill,
		SourceEntry:    request.SourceEntry,
		ServedByType:   servedByType,
		QueueEnteredAt: time.Now(),
		LastActivityAt: time.Now(),
	}
	center.sessions[request.VisitorID] = session
	return session
}

func (center *Center) assignLocked(session *sessionState, ownerID string, keepUnavailable bool) AssignmentResult {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return AssignmentResult{Assigned: false, RouteStatus: RouteStatusPending, QueueName: session.QueueName, Reason: "owner is required"}
	}
	if !keepUnavailable && !center.isKefuAvailableLocked(ownerID) {
		return AssignmentResult{Assigned: false, RouteStatus: RouteStatusPending, QueueName: session.QueueName, Reason: "客服当前不可接待"}
	}

	runtimeKefu := center.ensureKefuLocked(ownerID, "")
	displayName := runtimeKefu.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = ownerID
		if models.DB != nil {
			user := models.FindUser(ownerID)
			if strings.TrimSpace(user.Nickname) != "" {
				displayName = user.Nickname
			}
		}
	}
	runtimeKefu.DisplayName = displayName
	runtimeKefu.LastAssignedAt = time.Now()
	runtimeKefu.ActiveSessions = center.activeSessionCountLocked(ownerID)

	session.OwnerID = ownerID
	session.StickyOwnerID = ""
	session.RouteStatus = RouteStatusAssigned
	session.ServedByType = ServedByHuman
	session.LastRouteReason = ""
	session.QueueEnteredAt = time.Time{}
	session.LastAssignAttemptAt = time.Time{}
	session.LastAssignedAt = time.Now()
	session.LastActivityAt = time.Now()
	runtimeKefu.ActiveSessions = center.activeSessionCountLocked(ownerID)

	return AssignmentResult{
		Assigned:    true,
		OwnerID:     ownerID,
		DisplayName: displayName,
		RouteStatus: session.RouteStatus,
		QueueName:   session.QueueName,
	}
}

func (center *Center) ensureKefuLocked(kefuID string, displayName string) *RuntimeKefu {
	runtimeKefu, exists := center.kefus[kefuID]
	if exists {
		if strings.TrimSpace(displayName) != "" {
			runtimeKefu.DisplayName = displayName
		}
		return runtimeKefu
	}

	runtimeKefu = &RuntimeKefu{
		KefuID:            kefuID,
		DisplayName:       displayName,
		Skills:            models.GetUserRoutingSkills(kefuID),
		PresenceStatus:    PresenceOffline,
		AcceptingSessions: false,
		Enabled:           true,
		MaxSessions:       center.defaultMaxSessions,
		LastSeenAt:        time.Now(),
	}
	center.kefus[kefuID] = runtimeKefu
	return runtimeKefu
}

func (center *Center) markPendingLocked(session *sessionState, stickyOwnerID string, reason string) AssignmentResult {
	now := time.Now()
	if session.RouteStatus != RouteStatusPending || session.QueueEnteredAt.IsZero() {
		session.QueueEnteredAt = now
	}
	session.RouteStatus = RouteStatusPending
	session.OwnerID = ""
	session.LastRouteReason = reason
	session.LastAssignAttemptAt = now
	if strings.TrimSpace(stickyOwnerID) != "" {
		session.StickyOwnerID = stickyOwnerID
	}
	if session.PreferredSkill != "" {
		session.QueueName = session.PreferredSkill
	} else {
		session.QueueName = center.defaultQueueName
	}
	return AssignmentResult{
		Assigned:    false,
		OwnerID:     session.StickyOwnerID,
		RouteStatus: RouteStatusPending,
		QueueName:   session.QueueName,
		Reason:      reason,
	}
}

func (center *Center) processPendingSessionsLocked(now time.Time, force bool) (int, []SessionSnapshot) {
	processedCount := 0
	assignedSnapshots := make([]SessionSnapshot, 0)
	pendingSessions := make([]*sessionState, 0)
	for _, session := range center.sessions {
		if session.RouteStatus != RouteStatusPending {
			continue
		}
		pendingSessions = append(pendingSessions, session)
	}

	sort.SliceStable(pendingSessions, func(leftIndex, rightIndex int) bool {
		return pendingSessions[leftIndex].QueueEnteredAt.Before(pendingSessions[rightIndex].QueueEnteredAt)
	})

	for _, session := range pendingSessions {
		if !session.LastActivityAt.IsZero() && now.Sub(session.LastActivityAt) >= center.pendingTTL {
			session.RouteStatus = RouteStatusClosed
			session.LastRouteReason = "pending 会话长时间无活动，已自动回收"
			session.OwnerID = ""
			session.StickyOwnerID = ""
			continue
		}
		if !force && !session.LastAssignAttemptAt.IsZero() && now.Sub(session.LastAssignAttemptAt) < center.retryInterval {
			continue
		}

		if session.StickyOwnerID != "" && !session.QueueEnteredAt.IsZero() && now.Sub(session.QueueEnteredAt) < center.expandAfter {
			session.LastAssignAttemptAt = now
			if center.isKefuAvailableLocked(session.StickyOwnerID) {
				center.assignLocked(session, session.StickyOwnerID, false)
				assignedSnapshots = append(assignedSnapshots, cloneSession(session))
				processedCount++
				continue
			}
			session.LastRouteReason = "等待原客服恢复可接待"
			continue
		}

		if session.StickyOwnerID != "" && !session.QueueEnteredAt.IsZero() && now.Sub(session.QueueEnteredAt) >= center.expandAfter {
			session.StickyOwnerID = ""
			session.QueueName = center.defaultQueueName
			session.LastRouteReason = "等待超时，已扩散到公共队列"
		}
		if session.StickyOwnerID == "" && session.PreferredSkill != "" && session.QueueName != center.defaultQueueName && !session.QueueEnteredAt.IsZero() && now.Sub(session.QueueEnteredAt) >= center.expandAfter {
			session.QueueName = center.defaultQueueName
			session.LastRouteReason = "技能池等待超时，已扩散到公共队列"
		}

		routingSkill := session.PreferredSkill
		if session.QueueName == center.defaultQueueName {
			routingSkill = ""
		}
		selectedKefu := center.pickLeastLoadedKefuLocked(routingSkill)
		session.LastAssignAttemptAt = now
		if selectedKefu == nil {
			if routingSkill != "" {
				session.LastRouteReason = "当前技能池暂无可用客服"
			} else if session.LastRouteReason == "" {
				session.LastRouteReason = "暂无可用客服"
			}
			continue
		}
		center.assignLocked(session, selectedKefu.KefuID, false)
		assignedSnapshots = append(assignedSnapshots, cloneSession(session))
		processedCount++
	}

	for _, runtimeKefu := range center.kefus {
		runtimeKefu.ActiveSessions = center.activeSessionCountLocked(runtimeKefu.KefuID)
	}
	return processedCount, assignedSnapshots
}

func (center *Center) pickLeastLoadedKefuLocked(preferredSkill string) *RuntimeKefu {
	normalizedSkill := strings.ToLower(strings.TrimSpace(preferredSkill))
	candidates := make([]*RuntimeKefu, 0, len(center.kefus))
	for _, runtimeKefu := range center.kefus {
		runtimeKefu.ActiveSessions = center.activeSessionCountLocked(runtimeKefu.KefuID)
		if !center.isKefuAvailableLocked(runtimeKefu.KefuID) {
			continue
		}
		if normalizedSkill != "" && !runtimeKefu.hasSkill(normalizedSkill) {
			continue
		}
		candidates = append(candidates, runtimeKefu)
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(leftIndex, rightIndex int) bool {
		left := candidates[leftIndex]
		right := candidates[rightIndex]
		if left.ActiveSessions == right.ActiveSessions {
			return left.LastAssignedAt.Before(right.LastAssignedAt)
		}
		return left.ActiveSessions < right.ActiveSessions
	})

	return candidates[0]
}

func (runtimeKefu RuntimeKefu) hasSkill(skill string) bool {
	if skill == "" {
		return true
	}
	for _, currentSkill := range runtimeKefu.Skills {
		if strings.EqualFold(currentSkill, skill) {
			return true
		}
	}
	return false
}

func (center *Center) activeSessionCountLocked(kefuID string) int {
	activeCount := 0
	for _, session := range center.sessions {
		if session.OwnerID == kefuID && session.RouteStatus == RouteStatusAssigned {
			activeCount++
		}
	}
	return activeCount
}

func (center *Center) isKefuAvailableLocked(kefuID string) bool {
	runtimeKefu, exists := center.kefus[kefuID]
	if !exists {
		return false
	}
	if !runtimeKefu.Enabled || !runtimeKefu.AcceptingSessions || runtimeKefu.PresenceStatus != PresenceOnline {
		return false
	}
	return center.activeSessionCountLocked(kefuID) < runtimeKefu.MaxSessions
}

func isValidPresenceStatus(presenceStatus string) bool {
	switch strings.ToLower(strings.TrimSpace(presenceStatus)) {
	case PresenceOnline, PresenceAway, PresenceBusy, PresenceOffline:
		return true
	default:
		return false
	}
}

func cloneSession(session *sessionState) SessionSnapshot {
	return SessionSnapshot{
		VisitorID:           session.VisitorID,
		VisitorName:         session.VisitorName,
		OwnerID:             session.OwnerID,
		StickyOwnerID:       session.StickyOwnerID,
		RouteStatus:         session.RouteStatus,
		QueueName:           session.QueueName,
		PreferredSkill:      session.PreferredSkill,
		SourceEntry:         session.SourceEntry,
		ServedByType:        session.ServedByType,
		LastRouteReason:     session.LastRouteReason,
		QueueEnteredAt:      session.QueueEnteredAt,
		LastAssignAttemptAt: session.LastAssignAttemptAt,
		LastAssignedAt:      session.LastAssignedAt,
		LastTransferAt:      session.LastTransferAt,
		LastActivityAt:      session.LastActivityAt,
	}
}

func (center *Center) dispatchPendingAssigned(sessionSnapshots []SessionSnapshot) {
	if len(sessionSnapshots) == 0 || center.hooks.OnPendingAssigned == nil {
		return
	}
	for _, sessionSnapshot := range sessionSnapshots {
		center.hooks.OnPendingAssigned(sessionSnapshot)
	}
}
