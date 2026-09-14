// Package domain defines roster planning concepts and invariants.
package domain

import (
	"fmt"
	"time"
)

const (
	EffectRosterPenalty EffectKind = "roster-penalty"
	EffectFactorEvent   EffectKind = "factor-event"
)

type (
	// EventOccurrence is an immutable fact associated with a canonical member.
	// Its occupied interval is [StartsAt, EndsAt()).
	EventOccurrence struct {
		StartsAt   time.Time      `json:"starts_at"`
		Attributes map[string]any `json:"attributes,omitempty"`
		ID         string         `json:"id"`
		Type       string         `json:"type"`
		MemberID   string         `json:"member_id"`
		Duration   time.Duration  `json:"duration"`
	}

	// EffectKind identifies the distinct ways an event can influence planning.
	EffectKind string

	// Effect is a declarative result derived from an event.
	Effect struct {
		Kind          EffectKind `json:"kind"`
		SourceEventID string     `json:"source_event_id"`
		MemberID      string     `json:"member_id"`
		Role          string     `json:"role,omitempty"`
		Factor        string     `json:"factor,omitempty"`
		StartsAt      time.Time  `json:"starts_at"`
		EndsAt        time.Time  `json:"ends_at"`
		Reason        string     `json:"reason,omitempty"`
	}

	// Availability describes whether a member may be assigned and the cost of overriding that state.
	//nolint:govet // JSON field order follows the external availability contract.
	Availability struct {
		Available       bool    `json:"available"`
		NegotiationCost float64 `json:"negotiation_cost"`
		Reason          string  `json:"reason,omitempty"`
		Kind            string  `json:"kind,omitempty"`
	}
)

// EndsAt returns the exclusive upper bound of the occurrence interval.
func (event *EventOccurrence) EndsAt() time.Time {
	return event.StartsAt.Add(event.Duration)
}

// RosterPenalty creates a bounded roster penalty interval derived from an event.
func RosterPenalty(
	event *EventOccurrence,
	role string,
	startsAt time.Time,
	duration time.Duration,
	reason string,
) Effect {
	return Effect{
		Kind:          EffectRosterPenalty,
		SourceEventID: event.ID,
		MemberID:      event.MemberID,
		Role:          role,
		StartsAt:      startsAt,
		EndsAt:        startsAt.Add(duration),
		Reason:        reason,
	}
}

// Validate checks the bounded negotiation cost.
func (availability Availability) Validate() error {
	if availability.NegotiationCost < 0 || availability.NegotiationCost > 1 {
		return fmt.Errorf("negotiation cost %g is outside [0, 1]", availability.NegotiationCost)
	}
	return nil
}

// AssignmentBlocked reports whether the normalized availability is effectively locked.
func (availability Availability) AssignmentBlocked(threshold float64) bool {
	return !availability.Available && availability.NegotiationCost >= 1 ||
		availability.NegotiationCost >= threshold
}

// FactorEvent records a soft factor contribution caused by an event.
func FactorEvent(event *EventOccurrence, factor string, occurredAt time.Time) Effect {
	return Effect{
		Kind:          EffectFactorEvent,
		SourceEventID: event.ID,
		MemberID:      event.MemberID,
		Factor:        factor,
		StartsAt:      occurredAt,
		Reason:        "derived from " + event.Type,
	}
}
