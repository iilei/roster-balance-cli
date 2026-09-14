// Package domain defines roster planning concepts and invariants.
package domain

import "time"

const (
	EffectEligibilityLock EffectKind = "eligibility-lock"
	EffectFactorEvent     EffectKind = "factor-event"
)

type (
	// EventOccurrence is an immutable fact associated with a canonical member.
	// Its occupied interval is [StartsAt, EndsAt()).
	EventOccurrence struct {
		StartsAt   time.Time
		Attributes map[string]any
		ID         string
		Type       string
		MemberID   string
		Duration   time.Duration
	}

	// EffectKind identifies the distinct ways an event can influence planning.
	EffectKind string

	// Effect is a declarative result derived from an event.
	Effect struct {
		Kind          EffectKind
		SourceEventID string
		MemberID      string
		Role          string
		Factor        string
		StartsAt      time.Time
		EndsAt        time.Time
		Reason        string
	}
)

// EndsAt returns the exclusive upper bound of the occurrence interval.
func (event *EventOccurrence) EndsAt() time.Time {
	return event.StartsAt.Add(event.Duration)
}

// EligibilityLock creates a hard exclusion for a member and role.
func EligibilityLock(
	event *EventOccurrence,
	role string,
	startsAt time.Time,
	duration time.Duration,
	reason string,
) Effect {
	return Effect{
		Kind:          EffectEligibilityLock,
		SourceEventID: event.ID,
		MemberID:      event.MemberID,
		Role:          role,
		StartsAt:      startsAt,
		EndsAt:        startsAt.Add(duration),
		Reason:        reason,
	}
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
