package routing

import (
	"testing"
	"time"
)

func TestPendingSessionAssignedWhenKefuComesOnline(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   5 * time.Second,
		PendingTTL:    time.Minute,
	})

	result := center.AssignSession(AssignmentRequest{
		VisitorID:   "visitor-1",
		VisitorName: "Visitor One",
	})
	if result.Assigned {
		t.Fatalf("expected pending assignment when no kefu is online")
	}

	center.MarkKefuOnline("kefu-1", "Kefu One")

	sessionSnapshot, exists := center.GetSession("visitor-1")
	if !exists {
		t.Fatalf("expected session snapshot to exist after assignment")
	}
	if sessionSnapshot.RouteStatus != RouteStatusAssigned {
		t.Fatalf("expected session to become assigned after kefu online, got %s", sessionSnapshot.RouteStatus)
	}
	if sessionSnapshot.OwnerID != "kefu-1" {
		t.Fatalf("expected pending session assigned to kefu-1, got %s", sessionSnapshot.OwnerID)
	}
}

func TestStickyPendingSessionExpandsAfterTimeout(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   2 * time.Second,
		PendingTTL:    time.Minute,
	})

	result := center.AssignSession(AssignmentRequest{
		VisitorID:        "visitor-2",
		VisitorName:      "Visitor Two",
		PreferredOwnerID: "sticky-kefu",
		RequireAvailable: true,
		AllowStickyOwner: true,
	})
	if result.Assigned {
		t.Fatalf("expected pending assignment when sticky owner unavailable")
	}

	initialSnapshot, exists := center.GetSession("visitor-2")
	if !exists {
		t.Fatalf("expected session snapshot to exist")
	}

	center.MarkKefuOnline("backup-kefu", "Backup Kefu")
	center.ProcessPendingSessions(initialSnapshot.QueueEnteredAt.Add(1 * time.Second))

	beforeExpandSnapshot, _ := center.GetSession("visitor-2")
	if beforeExpandSnapshot.RouteStatus != RouteStatusPending {
		t.Fatalf("expected session to remain pending before expand timeout, got %s", beforeExpandSnapshot.RouteStatus)
	}
	if beforeExpandSnapshot.StickyOwnerID != "sticky-kefu" {
		t.Fatalf("expected sticky owner to remain before expand timeout, got %s", beforeExpandSnapshot.StickyOwnerID)
	}

	center.ProcessPendingSessions(initialSnapshot.QueueEnteredAt.Add(3 * time.Second))

	afterExpandSnapshot, _ := center.GetSession("visitor-2")
	if afterExpandSnapshot.RouteStatus != RouteStatusAssigned {
		t.Fatalf("expected session to be assigned after expand timeout, got %s", afterExpandSnapshot.RouteStatus)
	}
	if afterExpandSnapshot.OwnerID != "backup-kefu" {
		t.Fatalf("expected session reassigned to backup-kefu after expansion, got %s", afterExpandSnapshot.OwnerID)
	}
}

func TestPendingSessionAutoClosedAfterTTL(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   2 * time.Second,
		PendingTTL:    5 * time.Second,
	})

	result := center.AssignSession(AssignmentRequest{
		VisitorID:   "visitor-3",
		VisitorName: "Visitor Three",
	})
	if result.Assigned {
		t.Fatalf("expected pending assignment when no kefu is online")
	}

	sessionSnapshot, exists := center.GetSession("visitor-3")
	if !exists {
		t.Fatalf("expected session snapshot to exist")
	}

	center.ProcessPendingSessions(sessionSnapshot.LastActivityAt.Add(6 * time.Second))

	closedSnapshot, _ := center.GetSession("visitor-3")
	if closedSnapshot.RouteStatus != RouteStatusClosed {
		t.Fatalf("expected pending session to auto close after ttl, got %s", closedSnapshot.RouteStatus)
	}
}

func TestAssignSessionPrefersMatchingSkillPool(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   2 * time.Second,
		PendingTTL:    time.Minute,
	})

	salesKefu := center.ensureKefuLocked("sales-kefu", "Sales Kefu")
	salesKefu.PresenceStatus = PresenceOnline
	salesKefu.AcceptingSessions = true
	salesKefu.Enabled = true
	salesKefu.Skills = []string{"sales"}

	supportKefu := center.ensureKefuLocked("support-kefu", "Support Kefu")
	supportKefu.PresenceStatus = PresenceOnline
	supportKefu.AcceptingSessions = true
	supportKefu.Enabled = true
	supportKefu.Skills = []string{"support"}

	result := center.AssignSession(AssignmentRequest{
		VisitorID:      "visitor-4",
		VisitorName:    "Visitor Four",
		PreferredSkill: "support",
	})
	if !result.Assigned {
		t.Fatalf("expected assignment to succeed with matching skill pool")
	}
	if result.OwnerID != "support-kefu" {
		t.Fatalf("expected support skill visitor assigned to support-kefu, got %s", result.OwnerID)
	}
}

func TestPendingSkillSessionExpandsToPublicPool(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   2 * time.Second,
		PendingTTL:    time.Minute,
	})

	publicKefu := center.ensureKefuLocked("public-kefu", "Public Kefu")
	publicKefu.PresenceStatus = PresenceOnline
	publicKefu.AcceptingSessions = true
	publicKefu.Enabled = true
	publicKefu.Skills = []string{"general"}

	result := center.AssignSession(AssignmentRequest{
		VisitorID:      "visitor-5",
		VisitorName:    "Visitor Five",
		PreferredSkill: "refund",
	})
	if result.Assigned {
		t.Fatalf("expected refund visitor to stay pending when no matching skill exists")
	}

	initialSnapshot, _ := center.GetSession("visitor-5")
	center.ProcessPendingSessions(initialSnapshot.QueueEnteredAt.Add(3 * time.Second))

	expandedSnapshot, _ := center.GetSession("visitor-5")
	if expandedSnapshot.RouteStatus != RouteStatusAssigned {
		t.Fatalf("expected pending skill session assigned after expand timeout, got %s", expandedSnapshot.RouteStatus)
	}
	if expandedSnapshot.OwnerID != "public-kefu" {
		t.Fatalf("expected pending skill session assigned to public pool kefu, got %s", expandedSnapshot.OwnerID)
	}
}

func TestPendingAssignmentHookTriggeredAfterAutoDispatch(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   2 * time.Second,
		PendingTTL:    time.Minute,
	})

	triggeredVisitorID := ""
	triggeredOwnerID := ""
	center.SetHooks(Hooks{
		OnPendingAssigned: func(sessionSnapshot SessionSnapshot) {
			triggeredVisitorID = sessionSnapshot.VisitorID
			triggeredOwnerID = sessionSnapshot.OwnerID
		},
	})

	result := center.AssignSession(AssignmentRequest{
		VisitorID:      "visitor-6",
		VisitorName:    "Visitor Six",
		PreferredSkill: "support",
	})
	if result.Assigned {
		t.Fatalf("expected visitor to stay pending before matching kefu becomes available")
	}

	supportKefu := center.ensureKefuLocked("support-kefu", "Support Kefu")
	supportKefu.PresenceStatus = PresenceOnline
	supportKefu.AcceptingSessions = true
	supportKefu.Enabled = true
	supportKefu.Skills = []string{"support"}

	center.ProcessPendingSessions(time.Now().Add(3 * time.Second))

	if triggeredVisitorID != "visitor-6" {
		t.Fatalf("expected pending assignment hook visitor_id visitor-6, got %s", triggeredVisitorID)
	}
	if triggeredOwnerID != "support-kefu" {
		t.Fatalf("expected pending assignment hook owner_id support-kefu, got %s", triggeredOwnerID)
	}
}

func TestKefuRoutingStatusControlsAvailability(t *testing.T) {
	center := NewCenter(2, "default", AutoDispatchConfig{
		RetryInterval: time.Second,
		ExpandAfter:   2 * time.Second,
		PendingTTL:    time.Minute,
	})

	center.MarkKefuOnline("kefu-1", "Kefu One")
	_, exists := center.UpdateKefuRoutingStatus("kefu-1", PresenceAway, false)
	if !exists {
		t.Fatalf("expected kefu runtime state to exist")
	}

	result := center.AssignSession(AssignmentRequest{
		VisitorID:   "visitor-7",
		VisitorName: "Visitor Seven",
	})
	if result.Assigned {
		t.Fatalf("expected away kefu to be unavailable for new assignments")
	}

	center.UpdateKefuRoutingStatus("kefu-1", PresenceOnline, true)
	result = center.AssignSession(AssignmentRequest{
		VisitorID:   "visitor-8",
		VisitorName: "Visitor Eight",
	})
	if !result.Assigned {
		t.Fatalf("expected online accepting kefu to receive new assignments")
	}
}
