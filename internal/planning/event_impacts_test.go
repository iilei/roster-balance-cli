package planning_test

import (
	"testing"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/domain"
	dataio "github.com/iilei/roster-balance-cli/internal/io"
	"github.com/iilei/roster-balance-cli/internal/planning"
)

const (
	callRecoveryImpactMath = "call-recovery"
	onCallCallAnswered     = "on-call-call-answered"
	onCallDuty             = "on-call"
	memberOne              = "member-1"
	afterJoinOccurrenceID  = "after-join"
)

func TestResolveEventEffectsCreatesRosterLockFromOccurrenceEnd(t *testing.T) {
	const recoveryDuration = "24h"
	startsAt := time.Date(2026, 9, 14, 9, 15, 0, 0, time.UTC)
	event := &domain.EventOccurrence{
		ID:       "call-1",
		Type:     onCallCallAnswered,
		MemberID: memberOne,
		StartsAt: startsAt,
		Duration: 27 * time.Minute,
	}
	cfg := &config.Config{
		Limits: config.SystemLimits{MaxIrrelevantAfter: "26280h"},
		ImpactMaths: map[string]config.ImpactMath{
			callRecoveryImpactMath: {
				StartsFrom: "event.ends_at",
				Lifecycle: config.Lifecycle{
					DecayProfile:    "front-loaded",
					HoldDuration:    recoveryDuration,
					IrrelevantAfter: recoveryDuration,
				},
			},
		},
		EventTypes: map[string]config.EventType{
			onCallCallAnswered: {
				Impacts: []config.ImpactApplication{{
					ImpactMath: callRecoveryImpactMath,
					Effect:     "roster-penalty",
					Role:       onCallDuty,
				}},
			},
		},
	}

	effects, err := planning.ResolveEventEffects(cfg, event)
	if err != nil {
		t.Fatalf("ResolveEventEffects() error = %v", err)
	}
	if len(effects) != 1 {
		t.Fatalf("effect count = %d, want 1", len(effects))
	}
	if got, want := effects[0].StartsAt, event.EndsAt(); !got.Equal(want) {
		t.Fatalf("lock starts at %s, want occurrence end %s", got, want)
	}
	if got, want := effects[0].EndsAt, event.EndsAt().Add(24*time.Hour); !got.Equal(want) {
		t.Fatalf("lock ends at %s, want %s", got, want)
	}
}

func TestInspectRecommendationEvidenceUsesJoinedAtLookbackBoundary(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	member := dataio.TeamMember{
		MemberID:       memberOne,
		JoinedAt:       now.Add(-12 * time.Hour),
		EligibleDuties: []string{onCallDuty},
	}
	occurrences := []domain.EventOccurrence{
		{ID: "before-join", Type: onCallCallAnswered, MemberID: member.MemberID, StartsAt: now.Add(-13 * time.Hour)},
		{ID: afterJoinOccurrenceID, Type: onCallCallAnswered, MemberID: member.MemberID, StartsAt: now.Add(-time.Hour)},
	}
	cfg := &config.Config{Limits: config.SystemLimits{MaxIrrelevantAfter: "24h"}}

	evidence, err := planning.InspectRecommendationEvidence(
		cfg,
		onCallDuty,
		now,
		[]dataio.TeamMember{member},
		occurrences,
	)
	if err != nil {
		t.Fatalf("InspectRecommendationEvidence() error = %v", err)
	}
	candidate := evidence.Candidates[0]
	if !candidate.LookbackStart.Equal(member.JoinedAt) || candidate.LookbackBound != "joined_at" {
		t.Fatalf("lookback = (%s, %q), want joined-at boundary", candidate.LookbackStart, candidate.LookbackBound)
	}
	if len(candidate.Occurrences) != 1 || candidate.Occurrences[0].ID != afterJoinOccurrenceID {
		t.Fatalf("occurrences = %#v, want only occurrence after join", candidate.Occurrences)
	}
}
