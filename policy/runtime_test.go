package policy_test

import (
	"testing"
	"time"

	"github.com/iilei/roster-balance-cli/internal/domain"
	"github.com/iilei/roster-balance-cli/policy"
)

func TestRuntimeEvaluatesOnCallEffects(t *testing.T) {
	callAt := time.Date(2026, 9, 14, 3, 0, 0, 0, time.UTC)
	onCallEnds := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	event := domain.Event{
		ID:         "event-1",
		Type:       "on-call-call",
		MemberID:   "member-1",
		OccurredAt: callAt,
		Attributes: map[string]any{"received_at": callAt, "on_call_span": map[string]any{"ends_at": onCallEnds}},
	}
	source := `
def on_event(ctx):
    lock_start = time.max(ctx.event.received_at, ctx.event.on_call_span.ends_at)
    return [
        effects.eligibility_lock(
            role = "remediation-manager",
            starts_at = lock_start,
            duration_hours = 24,
            reason = "on-call recovery",
        ),
        effects.factor_event(
            factor = "duty-work-served",
        ),
    ]
`

	effects, err := (policy.Runtime{}).EvaluateEvent(&event, source)
	if err != nil {
		t.Fatalf("EvaluateEvent() error = %v", err)
	}
	if len(effects) != 2 {
		t.Fatalf("effect count = %d, want 2", len(effects))
	}
	if got := effects[0]; got.Kind != domain.EffectEligibilityLock || got.Role != "remediation-manager" ||
		!got.StartsAt.Equal(onCallEnds) ||
		!got.EndsAt.Equal(onCallEnds.Add(24*time.Hour)) {
		t.Fatalf("lock = %#v", got)
	}
	if got := effects[1]; got.Kind != domain.EffectFactorEvent || got.Factor != "duty-work-served" ||
		!got.StartsAt.Equal(callAt) {
		t.Fatalf("factor event = %#v", got)
	}
}
