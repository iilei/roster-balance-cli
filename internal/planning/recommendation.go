package planning

import (
	"errors"
	"fmt"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/domain"
	dataio "github.com/iilei/roster-balance-cli/internal/io"
)

type (
	// CandidateEvidence describes one duty-eligible member's relevant history.
	//nolint:govet // RFC 3339 timestamps keep this external JSON contract clear.
	CandidateEvidence struct {
		Occurrences   []domain.EventOccurrence `json:"occurrences"`
		Effects       []domain.Effect          `json:"effects"`
		ActiveLocks   []domain.Effect          `json:"active_locks"`
		MemberID      string                   `json:"member_id"`
		LookbackBound string                   `json:"lookback_bound"`
		JoinedAt      time.Time                `json:"joined_at"`
		LookbackStart time.Time                `json:"lookback_start"`
	}

	// RecommendationEvidence is the input for a future recommendation decision.
	//nolint:govet // The output preserves the planning instant as an RFC 3339 timestamp.
	RecommendationEvidence struct {
		Candidates []CandidateEvidence `json:"candidates"`
		Duty       string              `json:"duty"`
		Now        time.Time           `json:"now"`
		Lookback   time.Duration       `json:"lookback"`
	}

	lookbackBoundary struct {
		Start time.Time
		Bound string
	}
)

// InspectRecommendationEvidence returns bounded evidence for every duty-eligible candidate.
func InspectRecommendationEvidence(
	cfg *config.Config,
	duty string,
	now time.Time,
	members []dataio.TeamMember,
	occurrences []domain.EventOccurrence,
) (RecommendationEvidence, error) {
	maximum, err := config.ParseElapsedDuration(cfg.Limits.MaxIrrelevantAfter)
	if err != nil {
		return RecommendationEvidence{}, fmt.Errorf("max_irrelevant_after: %w", err)
	}
	if maximum <= 0 {
		return RecommendationEvidence{}, errors.New("max_irrelevant_after must be greater than zero")
	}

	result := RecommendationEvidence{
		Now:        now,
		Duty:       duty,
		Lookback:   maximum,
		Candidates: make([]CandidateEvidence, 0),
	}
	maximumStart := now.Add(-maximum)
	for index := range members {
		member := &members[index]
		if !member.EligibleForDuty(duty) || member.JoinedAt.After(now) {
			continue
		}
		candidate, err := inspectCandidateEvidence(cfg, member, occurrences, maximumStart, now)
		if err != nil {
			return RecommendationEvidence{}, err
		}
		result.Candidates = append(result.Candidates, candidate)
	}
	return result, nil
}

func inspectCandidateEvidence(
	cfg *config.Config,
	member *dataio.TeamMember,
	occurrences []domain.EventOccurrence,
	maximumStart, now time.Time,
) (CandidateEvidence, error) {
	candidate := CandidateEvidence{
		MemberID:    member.MemberID,
		JoinedAt:    member.JoinedAt,
		Occurrences: make([]domain.EventOccurrence, 0),
		Effects:     make([]domain.Effect, 0),
		ActiveLocks: make([]domain.Effect, 0),
	}
	boundary := lookbackStart(member.JoinedAt, maximumStart)
	candidate.LookbackStart = boundary.Start
	candidate.LookbackBound = boundary.Bound
	for occurrenceIndex := range occurrences {
		occurrence := &occurrences[occurrenceIndex]
		if occurrence.MemberID != member.MemberID ||
			!overlaps(occurrence.StartsAt, occurrence.EndsAt(), candidate.LookbackStart, now) {
			continue
		}
		candidate.Occurrences = append(candidate.Occurrences, *occurrence)
		effects, err := ResolveEventEffects(cfg, occurrence)
		if err != nil {
			return CandidateEvidence{}, fmt.Errorf("occurrence %q: %w", occurrence.ID, err)
		}
		appendRelevantEffects(&candidate, effects, now)
	}
	return candidate, nil
}

func appendRelevantEffects(candidate *CandidateEvidence, effects []domain.Effect, now time.Time) {
	for effectIndex := range effects {
		effect := &effects[effectIndex]
		if overlaps(effect.StartsAt, effect.EndsAt, candidate.LookbackStart, now) {
			candidate.Effects = append(candidate.Effects, *effect)
		}
		if activeAt(effect, now) {
			candidate.ActiveLocks = append(candidate.ActiveLocks, *effect)
		}
	}
}

func lookbackStart(joinedAt, maximumStart time.Time) lookbackBoundary {
	if joinedAt.After(maximumStart) {
		return lookbackBoundary{Start: joinedAt, Bound: "joined_at"}
	}
	return lookbackBoundary{Start: maximumStart, Bound: "max_irrelevant_after"}
}

func overlaps(startsAt, endsAt, from, until time.Time) bool {
	return startsAt.Before(until) && endsAt.After(from)
}

func activeAt(effect *domain.Effect, now time.Time) bool {
	return !effect.StartsAt.After(now) && effect.EndsAt.After(now)
}
