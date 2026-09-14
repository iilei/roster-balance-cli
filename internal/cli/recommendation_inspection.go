package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/domain"
	dataio "github.com/iilei/roster-balance-cli/internal/io"
	"github.com/iilei/roster-balance-cli/internal/planning"
	"github.com/iilei/roster-balance-cli/policy"
	"github.com/spf13/cobra"
)

const recommendationImpactSampleCount = 36

type (
	//nolint:govet // The output preserves the planning instant as an RFC 3339 timestamp.
	recommendationInspectionOutput struct {
		Candidates    []recommendationCandidateOutput `json:"candidates"`
		SchemaVersion string                          `json:"schema_version"`
		Duty          string                          `json:"duty"`
		Now           time.Time                       `json:"now"`
		LookbackHours float64                         `json:"lookback_hours"`
		FutureHours   float64                         `json:"future_hours"`
	}

	recommendationCandidateOutput struct {
		ImpactCurves    []impactCurve                `json:"impact_curves"`
		Timeline        []recommendationTimelineSpan `json:"timeline"`
		MemberID        string                       `json:"member_id"`
		LookbackBound   string                       `json:"lookback_bound"`
		JoinedAt        time.Time                    `json:"joined_at"`
		LookbackStart   time.Time                    `json:"lookback_start"`
		Occurrences     []domain.EventOccurrence     `json:"occurrences"`
		Effects         []domain.Effect              `json:"effects"`
		ActivePenalties []domain.Effect              `json:"active_penalties"`
	}

	recommendationTimelineSpan struct {
		Kind        string  `json:"kind"`
		Label       string  `json:"label"`
		StartsHours float64 `json:"starts_hours"`
		EndsHours   float64 `json:"ends_hours"`
	}

	impactCurve struct {
		ImpactMath  string              `json:"impact_math"`
		Description string              `json:"description"`
		Samples     []impactCurveSample `json:"samples"`
	}

	impactCurveSample struct {
		Hours  float64 `json:"hours"`
		Impact float64 `json:"impact"`
	}
)

func newInspectRecommendationCommand() *cobra.Command {
	var (
		duty             string
		nowText          = time.Now().UTC().Format(time.RFC3339)
		teamDataPaths    []string
		trackedDataPaths []string
		options          config.LoadOptions
	)

	cmd := &cobra.Command{
		Use:   "recommendation",
		Short: "Explain duty candidate evidence as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if duty == "" {
				return errors.New("duty is required")
			}
			if len(teamDataPaths) == 0 {
				return errors.New("team-data-fs is required")
			}
			if len(trackedDataPaths) == 0 {
				return errors.New("tracked-data-fs is required")
			}
			now, err := time.Parse(time.RFC3339, nowText)
			if err != nil {
				return fmt.Errorf("now must be an RFC 3339 instant: %w", err)
			}
			loaded, err := config.Load(options)
			if err != nil {
				return err
			}
			members, err := dataio.LoadTeamMembers(teamDataPaths)
			if err != nil {
				return fmt.Errorf("load team data: %w", err)
			}
			occurrences, err := dataio.LoadEventOccurrences(trackedDataPaths)
			if err != nil {
				return fmt.Errorf("load tracked data: %w", err)
			}
			evidence, err := planning.InspectRecommendationEvidence(&loaded, duty, now, members, occurrences)
			if err != nil {
				return err
			}
			output := recommendationInspectionOutput{
				SchemaVersion: "rosterbalance.recommendation-inspection/v1",
				Now:           evidence.Now,
				Duty:          evidence.Duty,
				LookbackHours: evidence.Lookback.Hours(),
				Candidates:    nil,
			}
			output.Candidates, err = recommendationCandidates(&loaded, evidence)
			if err != nil {
				return err
			}
			output.FutureHours = recommendationFutureHours(output.Candidates)
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		},
	}

	bindLoadFlags(cmd, &options)
	flags := cmd.Flags()
	flags.StringVar(&duty, "duty", "", "Duty to inspect")
	flags.StringVar(&nowText, "now", nowText, "RFC 3339 planning instant")
	flags.StringArrayVar(&teamDataPaths, "team-data-fs", nil, "External team data file; repeat for multiple files")
	flags.StringArrayVar(
		&trackedDataPaths,
		"tracked-data-fs",
		nil,
		"External tracked data file; repeat for multiple files",
	)
	return cmd
}

func recommendationCandidates(
	cfg *config.Config,
	evidence planning.RecommendationEvidence,
) ([]recommendationCandidateOutput, error) {
	candidates := make([]recommendationCandidateOutput, 0, len(evidence.Candidates))
	for index := range evidence.Candidates {
		candidate := &evidence.Candidates[index]
		output := recommendationCandidateOutput{
			MemberID:        candidate.MemberID,
			LookbackBound:   candidate.LookbackBound,
			JoinedAt:        candidate.JoinedAt,
			LookbackStart:   candidate.LookbackStart,
			Occurrences:     candidate.Occurrences,
			Effects:         candidate.Effects,
			ActivePenalties: candidate.ActivePenalties,
			Timeline:        make([]recommendationTimelineSpan, 0, len(candidate.Occurrences)+len(candidate.Effects)),
			ImpactCurves:    make([]impactCurve, 0, len(candidate.Effects)),
		}
		for occurrenceIndex := range candidate.Occurrences {
			occurrence := &candidate.Occurrences[occurrenceIndex]
			output.Timeline = append(output.Timeline, recommendationTimelineSpan{
				Kind:        "occurrence",
				Label:       occurrence.Type,
				StartsHours: occurrence.StartsAt.Sub(evidence.Now).Hours(),
				EndsHours:   occurrence.EndsAt().Sub(evidence.Now).Hours(),
			})
		}
		for effectIndex := range candidate.Effects {
			effect := &candidate.Effects[effectIndex]
			output.Timeline = append(output.Timeline, recommendationTimelineSpan{
				Kind:        string(effect.Kind),
				Label:       effect.Reason,
				StartsHours: effect.StartsAt.Sub(evidence.Now).Hours(),
				EndsHours:   effect.EndsAt.Sub(evidence.Now).Hours(),
			})
			curve, err := impactCurveForEffect(cfg, effect, evidence.Now)
			if err != nil {
				return nil, fmt.Errorf("effect from occurrence %q: %w", effect.SourceEventID, err)
			}
			output.ImpactCurves = append(output.ImpactCurves, curve)
		}
		candidates = append(candidates, output)
	}
	return candidates, nil
}

func impactCurveForEffect(cfg *config.Config, effect *domain.Effect, now time.Time) (impactCurve, error) {
	impactMath, ok := cfg.ImpactMaths[effect.Reason]
	if !ok {
		return impactCurve{}, fmt.Errorf("impact math %q not found", effect.Reason)
	}
	hold, err := config.ParseElapsedDuration(impactMath.Lifecycle.HoldDuration)
	if err != nil {
		return impactCurve{}, err
	}
	descriptor, ok := policy.DefaultDecayRegistry().Describe(impactMath.Lifecycle.DecayProfile)
	if !ok {
		return impactCurve{}, fmt.Errorf("decay profile %q not found", impactMath.Lifecycle.DecayProfile)
	}
	duration := effect.EndsAt.Sub(effect.StartsAt)
	curve := impactCurve{
		ImpactMath:  effect.Reason,
		Description: descriptor.Description,
		Samples:     make([]impactCurveSample, 0, recommendationImpactSampleCount+1),
	}
	for index := 0; index <= recommendationImpactSampleCount; index++ {
		elapsed := duration * time.Duration(index) / recommendationImpactSampleCount
		impact, err := impactAt(impactMath.Lifecycle.DecayProfile, elapsed, hold, duration)
		if err != nil {
			return impactCurve{}, err
		}
		curve.Samples = append(curve.Samples, impactCurveSample{
			Hours:  effect.StartsAt.Add(elapsed).Sub(now).Hours(),
			Impact: impact,
		})
	}
	return curve, nil
}

func recommendationFutureHours(candidates []recommendationCandidateOutput) float64 {
	futureHours := 0.0
	for candidateIndex := range candidates {
		candidate := &candidates[candidateIndex]
		for curveIndex := range candidate.ImpactCurves {
			curve := &candidate.ImpactCurves[curveIndex]
			if len(curve.Samples) == 0 {
				continue
			}
			futureHours = max(futureHours, curve.Samples[len(curve.Samples)-1].Hours)
		}
	}
	return futureHours
}

func impactAt(profile string, elapsed, hold, duration time.Duration) (float64, error) {
	if elapsed >= duration {
		return 0, nil
	}
	if duration == hold || elapsed < hold {
		return 1, nil
	}
	return policy.DefaultDecayRegistry().Evaluate(profile, float64(elapsed-hold)/float64(duration-hold))
}
