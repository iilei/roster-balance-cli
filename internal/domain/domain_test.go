// Package domain_test tests the domain package through its public API.
package domain_test

import (
	"testing"
	"time"

	"github.com/iilei/roster-balance-cli/internal/domain"
)

const (
	testEventID    = "event-1"
	testEventType  = "test"
	testMemberID   = "member-1"
	onCallCallType = "on-call-call"
)

func TestResolveLifecycleClampsFactorLifetime(t *testing.T) {
	resolved, err := domain.ResolveLifecycle(
		domain.Lifecycle{HoldDuration: 48 * time.Hour, IrrelevantAfter: 14 * 24 * time.Hour},
		domain.SystemLimits{MaxFactorLifetime: 7 * 24 * time.Hour},
	)
	if err != nil {
		t.Fatalf("ResolveLifecycle() error = %v", err)
	}
	if resolved.IrrelevantAfter != 7*24*time.Hour {
		t.Fatalf("irrelevant-after = %s, want 168h", resolved.IrrelevantAfter)
	}
}

func TestResolveLifecycleRejectsInvalidBoundaries(t *testing.T) {
	_, err := domain.ResolveLifecycle(
		domain.Lifecycle{HoldDuration: 48 * time.Hour, IrrelevantAfter: 24 * time.Hour},
		domain.SystemLimits{},
	)
	if err == nil {
		t.Fatal("ResolveLifecycle() error = nil, want invalid boundary error")
	}
}

func TestEffectsKeepHardAndSoftInfluencesDistinct(t *testing.T) {
	event := domain.Event{ID: testEventID, Type: onCallCallType, MemberID: testMemberID}
	lock := domain.EligibilityLock(&event, "remediation-manager", time.Unix(0, 0), 24*time.Hour, "on-call recovery")
	factor := domain.FactorEvent(&event, "duty-work-served", time.Unix(0, 0))

	if lock.Kind != domain.EffectEligibilityLock || lock.Role != "remediation-manager" || lock.Factor != "" {
		t.Fatalf("lock = %#v, want hard role lock", lock)
	}
	if factor.Kind != domain.EffectFactorEvent || factor.Factor != "duty-work-served" || factor.Role != "" {
		t.Fatalf("factor = %#v, want soft factor event", factor)
	}
}

func TestTrackRecordQueriesEventsAndOverlappingEffects(t *testing.T) {
	start := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	event := domain.Event{
		ID:         testEventID,
		Type:       onCallCallType,
		MemberID:   testMemberID,
		OccurredAt: start.Add(2 * time.Hour),
		Attributes: map[string]any{"role": "RB"},
	}
	lock := domain.EligibilityLock(
		&event,
		"remediation-manager",
		start.Add(3*time.Hour),
		24*time.Hour,
		"on-call recovery",
	)
	factor := domain.FactorEvent(&event, "duty-work-served", event.OccurredAt)

	var record domain.TrackRecord
	if err := record.Record(&event, lock, factor); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	events := record.EventsBetween(testMemberID, start, start.Add(3*time.Hour))
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("EventsBetween() = %#v, want event-1", events)
	}

	effects := record.EffectsOverlapping(testMemberID, start.Add(4*time.Hour), start.Add(5*time.Hour))
	if len(effects) != 1 || effects[0].Kind != domain.EffectEligibilityLock {
		t.Fatalf("EffectsOverlapping() = %#v, want active lock only", effects)
	}
}

func TestTrackRecordRejectsDuplicateAndMismatchedEffects(t *testing.T) {
	event := domain.Event{ID: testEventID, Type: testEventType, MemberID: testMemberID, OccurredAt: time.Now()}
	wrongEvent := domain.Event{ID: "event-2", Type: testEventType, MemberID: "member-2", OccurredAt: event.OccurredAt}
	lock := domain.EligibilityLock(&wrongEvent, "role", event.OccurredAt, time.Hour, "wrong source")

	var record domain.TrackRecord
	if err := record.Record(&event, lock); err == nil {
		t.Fatal("Record() error = nil, want source mismatch")
	}
	if err := record.Record(&event); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if err := record.Record(&event); err == nil {
		t.Fatal("Record() duplicate error = nil")
	}
}
