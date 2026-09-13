package domain

import (
	"testing"
	"time"
)

func TestResolveLifecycleClampsFactorLifetime(t *testing.T) {
	resolved, err := ResolveLifecycle(
		Lifecycle{HoldDuration: 48 * time.Hour, IrrelevantAfter: 14 * 24 * time.Hour},
		SystemLimits{MaxFactorLifetime: 7 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("ResolveLifecycle() error = %v", err)
	}
	if resolved.IrrelevantAfter != 7*24*time.Hour {
		t.Fatalf("irrelevant-after = %s, want 168h", resolved.IrrelevantAfter)
	}
}

func TestResolveLifecycleRejectsInvalidBoundaries(t *testing.T) {
	_, err := ResolveLifecycle(
		Lifecycle{HoldDuration: 48 * time.Hour, IrrelevantAfter: 24 * time.Hour},
		SystemLimits{},
	)
	if err == nil {
		t.Fatal("ResolveLifecycle() error = nil, want invalid boundary error")
	}
}

func TestEffectsKeepHardAndSoftInfluencesDistinct(t *testing.T) {
	event := Event{ID: "event-1", Type: "on-call-call", MemberID: "member-1"}
	lock := EligibilityLock(event, "remediation-manager", time.Unix(0, 0), 24*time.Hour, "on-call recovery")
	factor := FactorEvent(event, "duty-work-served", time.Unix(0, 0))

	if lock.Kind != EffectEligibilityLock || lock.Role != "remediation-manager" || lock.Factor != "" {
		t.Fatalf("lock = %#v, want hard role lock", lock)
	}
	if factor.Kind != EffectFactorEvent || factor.Factor != "duty-work-served" || factor.Role != "" {
		t.Fatalf("factor = %#v, want soft factor event", factor)
	}
}

func TestTrackRecordQueriesEventsAndOverlappingEffects(t *testing.T) {
	start := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	event := Event{
		ID:         "event-1",
		Type:       "on-call-call",
		MemberID:   "member-1",
		OccurredAt: start.Add(2 * time.Hour),
		Attributes: map[string]any{"role": "RB"},
	}
	lock := EligibilityLock(event, "remediation-manager", start.Add(3*time.Hour), 24*time.Hour, "on-call recovery")
	factor := FactorEvent(event, "duty-work-served", event.OccurredAt)

	var record TrackRecord
	if err := record.Record(event, lock, factor); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	events := record.EventsBetween("member-1", start, start.Add(3*time.Hour))
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("EventsBetween() = %#v, want event-1", events)
	}

	effects := record.EffectsOverlapping("member-1", start.Add(4*time.Hour), start.Add(5*time.Hour))
	if len(effects) != 1 || effects[0].Kind != EffectEligibilityLock {
		t.Fatalf("EffectsOverlapping() = %#v, want active lock only", effects)
	}
}

func TestTrackRecordRejectsDuplicateAndMismatchedEffects(t *testing.T) {
	event := Event{ID: "event-1", Type: "test", MemberID: "member-1", OccurredAt: time.Now()}
	wrongEvent := Event{ID: "event-2", Type: "test", MemberID: "member-2", OccurredAt: event.OccurredAt}
	lock := EligibilityLock(wrongEvent, "role", event.OccurredAt, time.Hour, "wrong source")

	var record TrackRecord
	if err := record.Record(event, lock); err == nil {
		t.Fatal("Record() error = nil, want source mismatch")
	}
	if err := record.Record(event); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if err := record.Record(event); err == nil {
		t.Fatal("Record() duplicate error = nil")
	}
}
