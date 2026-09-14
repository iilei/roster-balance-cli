package planning_test

import (
	"testing"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/domain"
	"github.com/iilei/roster-balance-cli/internal/planning"
)

const (
	callRecoveryImpactMath = "call-recovery"
	onCallCallAnswered     = "on-call-call-answered"
)

func TestResolveEventEffectsCreatesRosterLockFromOccurrenceEnd(t *testing.T) {
	const recoveryDuration = "24h"
	startsAt := time.Date(2026, 9, 14, 9, 15, 0, 0, time.UTC)
	event := &domain.EventOccurrence{
		ID:       "call-1",
		Type:     onCallCallAnswered,
		MemberID: "member-1",
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
					Effect:     "roster-lock",
					Role:       "on-call",
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
