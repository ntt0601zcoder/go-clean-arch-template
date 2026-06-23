package domain

import "time"

// AccountEventType classifies a domain event emitted when an account changes.
type AccountEventType string

const (
	AccountEventCreated AccountEventType = "account.created"
	AccountEventUpdated AccountEventType = "account.updated"
	AccountEventDeleted AccountEventType = "account.deleted"
)

// AccountEvent is a generic domain event the worker consumes off Kafka. It is a
// plain notification ("this account changed"), not a change-data-capture/sync
// payload — the template uses it only to demonstrate typed, at-least-once
// asynchronous processing.
type AccountEvent struct {
	Type       AccountEventType `json:"type"`
	AccountID  string           `json:"account_id"`
	OccurredAt time.Time        `json:"occurred_at"`
}
