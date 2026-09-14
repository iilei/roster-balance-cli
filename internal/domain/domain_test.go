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

func TestResolveLifecycleAllowsEqualBoundaries(t *testing.T) {
	const cutoff = 24 * time.Hour
	resolved, err := domain.ResolveLifecycle(
		domain.Lifecycle{HoldDuration: cutoff, IrrelevantAfter: cutoff},
		domain.SystemLimits{},
	)
	if err != nil {
		t.Fatalf("ResolveLifecycle() error = %v", err)
	}
	if resolved.HoldDuration != cutoff || resolved.IrrelevantAfter != cutoff {
		t.Fatalf("ResolveLifecycle() = %#v, want equal 24h boundaries", resolved)
	}
}

func TestEffectsKeepHardAndSoftInfluencesDistinct(t *testing.T) {
	event := domain.EventOccurrence{ID: testEventID, Type: onCallCallType, MemberID: testMemberID}
	lock := domain.RosterPenalty(&event, "remediation-manager", time.Unix(0, 0), 24*time.Hour, "on-call recovery")
	factor := domain.FactorEvent(&event, "duty-work-served", time.Unix(0, 0))

	if lock.Kind != domain.EffectRosterPenalty || lock.Role != "remediation-manager" || lock.Factor != "" {
		t.Fatalf("lock = %#v, want hard role lock", lock)
	}
	if factor.Kind != domain.EffectFactorEvent || factor.Factor != "duty-work-served" || factor.Role != "" {
		t.Fatalf("factor = %#v, want soft factor event", factor)
	}
}

func TestTrackRecordQueriesEventsAndOverlappingEffects(t *testing.T) {
	start := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	event := domain.EventOccurrence{
		ID:         testEventID,
		Type:       onCallCallType,
		MemberID:   testMemberID,
		StartsAt:   start.Add(2 * time.Hour),
		Duration:   time.Hour,
		Attributes: map[string]any{"role": "RB"},
	}
	lock := domain.RosterPenalty(
		&event,
		"remediation-manager",
		start.Add(3*time.Hour),
		24*time.Hour,
		"on-call recovery",
	)
	factor := domain.FactorEvent(&event, "duty-work-served", event.StartsAt)

	var record domain.TrackRecord
	if err := record.Record(&event, lock, factor); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	events := record.EventsBetween(testMemberID, start, start.Add(3*time.Hour))
	if len(events) != 1 || events[0].ID != event.ID {
		t.Fatalf("EventsBetween() = %#v, want event-1", events)
	}

	effects := record.EffectsOverlapping(testMemberID, start.Add(4*time.Hour), start.Add(5*time.Hour))
	if len(effects) != 1 || effects[0].Kind != domain.EffectRosterPenalty {
		t.Fatalf("EffectsOverlapping() = %#v, want active lock only", effects)
	}
}

func TestTrackRecordRejectsDuplicateAndMismatchedEffects(t *testing.T) {
	event := domain.EventOccurrence{ID: testEventID, Type: testEventType, MemberID: testMemberID, StartsAt: time.Now()}
	wrongEvent := domain.EventOccurrence{
		ID:       "event-2",
		Type:     testEventType,
		MemberID: "member-2",
		StartsAt: event.StartsAt,
	}
	lock := domain.RosterPenalty(&wrongEvent, "role", event.StartsAt, time.Hour, "wrong source")

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

func TestEventOccurrenceUsesAnExclusiveEnd(t *testing.T) {
	start := time.Date(2026, 9, 13, 9, 15, 0, 0, time.UTC)
	event := domain.EventOccurrence{StartsAt: start, Duration: 27 * time.Minute}
	if got, want := event.EndsAt(), start.Add(27*time.Minute); !got.Equal(want) {
		t.Fatalf("EndsAt() = %s, want %s", got, want)
	}
}

func TestAvailabilityUsesCostAndThreshold(t *testing.T) {
	cases := []struct {
		name      string
		available bool
		blocked   bool
		cost      float64
		threshold float64
	}{
		{name: "categorically unavailable", available: false, cost: 1, threshold: 1, blocked: true},
		{name: "fallback unavailable", available: false, cost: 0.5, threshold: 1, blocked: false},
		{name: "freely available", available: true, cost: 0, threshold: 1, blocked: false},
		{name: "penalty threshold", available: true, cost: 1, threshold: 1, blocked: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			availability := domain.Availability{Available: testCase.available, NegotiationCost: testCase.cost}
			if err := availability.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got := availability.AssignmentBlocked(testCase.threshold); got != testCase.blocked {
				t.Fatalf("AssignmentBlocked() = %v, want %v", got, testCase.blocked)
			}
		})
	}
}
