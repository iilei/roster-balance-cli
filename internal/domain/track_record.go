package domain

import (
	"errors"
	"fmt"
	"maps"
	"time"
)

// TrackRecord stores immutable events and their declarative derived effects.
type TrackRecord struct {
	events  []EventOccurrence
	effects []Effect
}

// Record appends an event and effects derived from that event.
func (record *TrackRecord) Record(event *EventOccurrence, effects ...Effect) error {
	if event.ID == "" {
		return errors.New("event ID is required")
	}
	if event.Type == "" {
		return errors.New("event type is required")
	}
	if event.MemberID == "" {
		return errors.New("event member ID is required")
	}
	if event.StartsAt.IsZero() {
		return errors.New("event starts-at time is required")
	}
	if event.Duration < 0 {
		return errors.New("event duration must not be negative")
	}
	for _, existing := range record.events {
		if existing.ID == event.ID {
			return fmt.Errorf("event %q is already recorded", event.ID)
		}
	}

	for index := range effects {
		effect := &effects[index]
		if effect.SourceEventID != event.ID {
			return fmt.Errorf("effect source event %q does not match event %q", effect.SourceEventID, event.ID)
		}
		if effect.MemberID != event.MemberID {
			return fmt.Errorf("effect member %q does not match event member %q", effect.MemberID, event.MemberID)
		}
		if !effect.EndsAt.IsZero() && effect.EndsAt.Before(effect.StartsAt) {
			return errors.New("effect ends-at time must not precede starts-at time")
		}
	}

	record.events = append(record.events, copyEvent(event))
	record.effects = append(record.effects, effects...)
	return nil
}

// EventsBetween returns occurrences overlapping the half-open interval [from, until).
func (record *TrackRecord) EventsBetween(memberID string, from, until time.Time) []EventOccurrence {
	result := make([]EventOccurrence, 0)
	for index := range record.events {
		event := &record.events[index]
		if event.MemberID == memberID && event.StartsAt.Before(until) && event.EndsAt().After(from) {
			result = append(result, copyEvent(event))
		}
	}
	return result
}

// EffectsOverlapping returns effects that can influence the half-open interval [from, until).
func (record *TrackRecord) EffectsOverlapping(memberID string, from, until time.Time) []Effect {
	result := make([]Effect, 0)
	for index := range record.effects {
		effect := &record.effects[index]
		if effect.MemberID != memberID || effect.StartsAt.After(until) {
			continue
		}
		if effect.EndsAt.IsZero() {
			if inHalfOpenRange(effect.StartsAt, from, until) {
				result = append(result, *effect)
			}
			continue
		}
		if effect.EndsAt.After(from) && effect.StartsAt.Before(until) {
			result = append(result, *effect)
		}
	}
	return result
}

func inHalfOpenRange(value, from, until time.Time) bool {
	return !value.Before(from) && value.Before(until)
}

func copyEvent(event *EventOccurrence) EventOccurrence {
	eventCopy := *event
	eventCopy.Attributes = make(map[string]any, len(event.Attributes))
	maps.Copy(eventCopy.Attributes, event.Attributes)
	return eventCopy
}
