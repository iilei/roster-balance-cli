package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/policy"
)

const (
	hoursPerDay = 24
	sampleCount = 48
)

type (
	factorInspectionOutput struct {
		SchemaVersion string                 `json:"schema_version"`
		Factors       []factorInspectionItem `json:"factors"`
	}

	factorInspectionItem struct {
		Name      string              `json:"name"`
		Lifecycle lifecycleInspection `json:"lifecycle"`
	}

	lifecycleInspection struct {
		Curve                decayProfileExplanation `json:"curve"`
		HoldDurationHours    float64                 `json:"hold_duration_hours"`
		IrrelevantAfterHours float64                 `json:"irrelevant_after_hours"`
	}

	decayProfileExplanation struct {
		Reference   string         `json:"reference"`
		Origin      string         `json:"origin"`
		Language    string         `json:"language"`
		Source      string         `json:"source,omitempty"`
		Description string         `json:"description,omitempty"`
		Samples     []impactSample `json:"samples"`
	}

	impactSample struct {
		X      float64 `json:"x"`
		Impact float64 `json:"impact"`
	}
)

func inspectFactors(cfg *config.Config) (factorInspectionOutput, error) {
	output := factorInspectionOutput{
		SchemaVersion: "rosterbalance.factor-output/v1",
		Factors:       make([]factorInspectionItem, 0, len(cfg.Factors)),
	}
	for _, factor := range cfg.Factors {
		hold, err := parseLifecycleDuration(factor.Lifecycle.HoldDuration)
		if err != nil {
			return factorInspectionOutput{}, fmt.Errorf("factor %q hold_duration: %w", factor.Name, err)
		}
		irrelevantAfter, err := parseLifecycleDuration(factor.Lifecycle.IrrelevantAfter)
		if err != nil {
			return factorInspectionOutput{}, fmt.Errorf("factor %q irrelevant_after: %w", factor.Name, err)
		}
		explanation, err := buildDecayProfileExplanation(
			factor.Lifecycle.DecayProfile,
			hold.Hours(),
			irrelevantAfter.Hours(),
		)
		if err != nil {
			return factorInspectionOutput{}, fmt.Errorf("factor %q: %w", factor.Name, err)
		}
		output.Factors = append(output.Factors, factorInspectionItem{
			Name: factor.Name,
			Lifecycle: lifecycleInspection{
				HoldDurationHours:    hold.Hours(),
				IrrelevantAfterHours: irrelevantAfter.Hours(),
				Curve:                explanation,
			},
		})
	}
	return output, nil
}

func buildDecayProfileExplanation(
	profile string,
	holdHours, irrelevantAfterHours float64,
) (decayProfileExplanation, error) {
	if holdHours > irrelevantAfterHours {
		return decayProfileExplanation{}, fmt.Errorf(
			"hold_duration_hours (%g) must not exceed irrelevant_after_hours (%g)",
			holdHours,
			irrelevantAfterHours,
		)
	}

	registry := policy.DefaultDecayRegistry()
	descriptor, ok := registry.Describe(profile)
	if !ok {
		return decayProfileExplanation{}, fmt.Errorf("unsupported decay profile %q", profile)
	}

	explanation := decayProfileExplanation{
		Reference:   descriptor.Reference,
		Origin:      descriptor.Origin,
		Language:    descriptor.Language,
		Source:      descriptor.Source,
		Description: descriptor.Description,
		Samples:     make([]impactSample, 0, sampleCount+1),
	}
	holdX := holdHours / irrelevantAfterHours
	for index := 0; index <= sampleCount; index++ {
		x := float64(index) / sampleCount
		impact, err := lifecycleImpact(registry, profile, x, holdX, index)
		if err != nil {
			return decayProfileExplanation{}, err
		}
		explanation.Samples = append(explanation.Samples, impactSample{X: x, Impact: impact})
	}
	return explanation, nil
}

func lifecycleImpact(
	registry *policy.DecayRegistry,
	profile string,
	x, holdX float64,
	index int,
) (float64, error) {
	if index == sampleCount {
		return 0, nil
	}
	if holdX == 1 || x < holdX {
		return 1, nil
	}
	decayX := (x - holdX) / (1 - holdX)
	impact, err := registry.Evaluate(profile, decayX)
	if err != nil {
		return 0, fmt.Errorf("profile %q: %w", profile, err)
	}
	if impact < 0 || impact > 1 {
		return 0, fmt.Errorf("profile %q returned out-of-range impact %g", profile, impact)
	}
	return impact, nil
}

func parseLifecycleDuration(value string) (time.Duration, error) {
	if daysText, ok := strings.CutSuffix(value, "d"); ok {
		days, err := strconv.ParseFloat(daysText, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days * float64(hoursPerDay*time.Hour)), nil
	}
	return time.ParseDuration(value)
}
