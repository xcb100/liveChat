package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"goflylivechat/models"
	"goflylivechat/tools"
)

const (
	EventMessageCreated       = "message.created"
	EventSessionAssigned      = "session.assigned"
	EventSessionTransferred   = "session.transferred"
	EventSessionClosed        = "session.closed"
	EventSessionPendingExpand = "session.pending.expanded"
)

type WorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  uint
}

type eventPayload struct {
	VisitorID string `json:"visitor_id"`
	SessionID string `json:"session_id"`
}

func EnqueueSessionScopedEvent(eventType string, aggregateType string, visitorID string) {
	if visitorID == "" {
		return
	}
	aggregateID := visitorID
	sessionID := ""
	if conversationSession := models.FindLatestConversationSessionByVisitorID(visitorID); conversationSession.ID != 0 {
		sessionID = conversationSession.SessionID
		if sessionID != "" {
			aggregateID = sessionID
		}
	}
	EnqueueBestEffort(eventType, aggregateType, aggregateID, eventPayload{
		VisitorID: visitorID,
		SessionID: sessionID,
	})
}

func EnqueueBestEffort(eventType string, aggregateType string, aggregateID string, payload interface{}) {
	if eventType == "" || aggregateType == "" {
		return
	}
	payloadBytes, marshalError := json.Marshal(payload)
	if marshalError != nil {
		tools.Logger().Warnf("outbox enqueue marshal failed event_type=%s aggregate_id=%s err=%v", eventType, aggregateID, marshalError)
		return
	}

	eventID := models.CreateOutboxEvent(&models.OutboxEvent{
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Payload:       string(payloadBytes),
		Status:        models.OutboxStatusPending,
	})
	if eventID == 0 {
		tools.Logger().Warnf("outbox enqueue failed event_type=%s aggregate_id=%s", eventType, aggregateID)
	}
}

func StartWorker(ctx context.Context, workerConfig WorkerConfig) {
	if workerConfig.PollInterval <= 0 {
		workerConfig.PollInterval = 2 * time.Second
	}
	if workerConfig.BatchSize <= 0 {
		workerConfig.BatchSize = 20
	}
	if workerConfig.MaxAttempts == 0 {
		workerConfig.MaxAttempts = 5
	}

	go func() {
		ticker := time.NewTicker(workerConfig.PollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processBatch(workerConfig)
			}
		}
	}()
}

func processBatch(workerConfig WorkerConfig) {
	pendingEvents := models.FindPendingOutboxEvents(workerConfig.BatchSize, time.Now(), workerConfig.MaxAttempts)
	for _, pendingEvent := range pendingEvents {
		if !models.MarkOutboxEventProcessing(pendingEvent.ID) {
			continue
		}
		models.IncrementOutboxEventAttempts(pendingEvent.ID)
		if handleError := handleEvent(pendingEvent); handleError != nil {
			nextRetryAt := time.Now().Add(time.Duration(pendingEvent.Attempts+1) * workerConfig.PollInterval)
			models.MarkOutboxEventFailed(pendingEvent.ID, pendingEvent.Attempts+1, handleError.Error(), nextRetryAt)
			tools.Logger().Warnf("outbox handle failed id=%d event_type=%s attempt=%d err=%v", pendingEvent.ID, pendingEvent.EventType, pendingEvent.Attempts+1, handleError)
			continue
		}
		models.MarkOutboxEventPublished(pendingEvent.ID)
		tools.Logger().Infof("outbox published id=%d event_type=%s aggregate_id=%s", pendingEvent.ID, pendingEvent.EventType, pendingEvent.AggregateID)
	}
}

func handleEvent(outboxEvent models.OutboxEvent) error {
	payload := eventPayload{}
	if parseError := json.Unmarshal([]byte(outboxEvent.Payload), &payload); parseError != nil {
		return parseError
	}

	switch outboxEvent.EventType {
	case EventMessageCreated, EventSessionAssigned, EventSessionTransferred, EventSessionClosed, EventSessionPendingExpand:
		if payload.VisitorID == "" {
			return errors.New("visitor_id is required")
		}
		models.SyncSessionSummaryByVisitorID(payload.VisitorID)
		return nil
	default:
		return errors.New("unsupported outbox event type")
	}
}
