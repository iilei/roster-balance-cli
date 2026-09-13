// Package cli defines the rosterbalance command-line interface.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/planning"
	"github.com/iilei/roster-balance-cli/policy"
	"github.com/spf13/cobra"
)

const (
	flatDecayProfile = "flat"
	hoursPerDay      = 24
)

type (
	// Version contains the build metadata injected at build time.
	Version struct {
		Version string
		Commit  string
		Date    string
	}

	factorOutput struct {
		SchemaVersion string             `json:"schema_version"`
		Factors       []factorOutputItem `json:"factors"`
	}

	factorOutputItem struct {
		Name      string          `json:"name"`
		Lifecycle lifecycleOutput `json:"lifecycle"`
	}

	lifecycleOutput struct {
		DecayProfile         string           `json:"decay_profile"`
		Curve                decayCurveOutput `json:"curve"`
		HoldDurationHours    float64          `json:"hold_duration_hours"`
		IrrelevantAfterHours float64          `json:"irrelevant_after_hours"`
	}

	decayCurveOutput struct {
		Reference   string       `json:"reference"`
		Origin      string       `json:"origin"`
		Language    string       `json:"language"`
		Source      string       `json:"source,omitempty"`
		Description string       `json:"description,omitempty"`
		Kind        string       `json:"kind"`
		Samples     []curvePoint `json:"samples"`
		Exponent    float64      `json:"exponent,omitempty"`
	}

	curvePoint struct {
		X      float64 `json:"x"`
		Impact float64 `json:"impact"`
	}
)

// NewRootCommand returns the CLI root command.
func NewRootCommand(version Version) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "rosterbalance",
		Short: "Plan roster assignments with validated config",
		Long:  "rosterbalance plans roster assignments from a validated config file and command-line overrides.",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built at: %s)", version.Version, version.Commit, version.Date)
	rootCmd.AddCommand(newValidateCommand())
	rootCmd.AddCommand(newPlanCommand())
	rootCmd.AddCommand(newInspectCommand())

	return rootCmd
}

func newInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect validated configuration data",
	}
	cmd.AddCommand(newInspectFactorsCommand())
	return cmd
}

func newInspectFactorsCommand() *cobra.Command {
	var options config.LoadOptions

	cmd := &cobra.Command{
		Use:   "factors",
		Short: "Print factor lifecycles as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			loaded, err := config.Load(options)
			if err != nil {
				return err
			}

			output := factorOutput{
				SchemaVersion: "rosterbalance.factor-output/v1",
				Factors:       make([]factorOutputItem, 0, len(loaded.Factors)),
			}
			for _, factor := range loaded.Factors {
				hold, err := parseLifecycleDuration(factor.Lifecycle.HoldDuration)
				if err != nil {
					return fmt.Errorf("factor %q hold_duration: %w", factor.Name, err)
				}
				irrelevantAfter, err := parseLifecycleDuration(factor.Lifecycle.IrrelevantAfter)
				if err != nil {
					return fmt.Errorf("factor %q irrelevant_after: %w", factor.Name, err)
				}
				curve, err := buildDecayCurve(factor.Lifecycle.DecayProfile, hold.Hours(), irrelevantAfter.Hours())
				if err != nil {
					return fmt.Errorf("factor %q: %w", factor.Name, err)
				}
				output.Factors = append(output.Factors, factorOutputItem{
					Name: factor.Name,
					Lifecycle: lifecycleOutput{
						DecayProfile:         factor.Lifecycle.DecayProfile,
						HoldDurationHours:    hold.Hours(),
						IrrelevantAfterHours: irrelevantAfter.Hours(),
						Curve:                curve,
					},
				})
			}

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		},
	}

	bindLoadFlags(cmd, &options)
	return cmd
}

func buildDecayCurve(profile string, holdHours, irrelevantAfterHours float64) (decayCurveOutput, error) {
	const sampleCount = 48

	if holdHours >= irrelevantAfterHours {
		return decayCurveOutput{}, fmt.Errorf(
			"hold_duration_hours (%g) must be less than irrelevant_after_hours (%g)",
			holdHours,
			irrelevantAfterHours,
		)
	}

	registry := policy.DefaultDecayRegistry()
	desc, ok := registry.Describe(profile)
	if !ok {
		return decayCurveOutput{}, fmt.Errorf("unsupported decay profile %q", profile)
	}

	curve := decayCurveOutput{
		Reference:   desc.Reference,
		Origin:      desc.Origin,
		Language:    desc.Language,
		Source:      desc.Source,
		Description: desc.Description,
		Kind:        "power",
		Samples:     make([]curvePoint, 0, sampleCount+1),
	}
	switch profile {
	case flatDecayProfile:
		curve.Kind = "step"
	case "front-loaded":
		curve.Exponent = 0.1
	case "linear":
		curve.Exponent = 1
	case "back-loaded":
		curve.Exponent = 4
	}

	// x is normalized over the full lifecycle; decay only starts after the hold span.
	holdX := holdHours / irrelevantAfterHours
	for index := 0; index <= sampleCount; index++ {
		x := float64(index) / sampleCount
		impact := 0.0
		switch {
		case x < holdX:
			impact = 1
		case profile == flatDecayProfile:
			if index < sampleCount {
				impact = 1
			}
		default:
			decayX := (x - holdX) / (1 - holdX)
			var err error
			impact, err = registry.Evaluate(profile, decayX)
			if err != nil {
				return decayCurveOutput{}, fmt.Errorf("profile %q: %w", profile, err)
			}
			if impact < 0 || impact > 1 {
				return decayCurveOutput{}, fmt.Errorf("profile %q returned out-of-range impact %g", profile, impact)
			}
		}
		curve.Samples = append(curve.Samples, curvePoint{X: x, Impact: impact})
	}
	return curve, nil
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

func newValidateCommand() *cobra.Command {
	var options config.LoadOptions

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the effective config",
		Long:  "Validate the effective rosterbalance config after defaults, file discovery, and overrides are applied.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("validate does not accept positional arguments")
			}

			_, err := config.Load(options)
			if err != nil {
				return err
			}

			cmd.Println("config valid")
			return nil
		},
	}

	bindLoadFlags(cmd, &options)
	return cmd
}

func newPlanCommand() *cobra.Command {
	var options config.LoadOptions

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan roster assignments",
		Long:  "Plan roster assignments from the effective config without requiring positional arguments.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return errors.New("plan does not accept positional arguments")
			}

			loaded, err := config.Load(options)
			if err != nil {
				return err
			}

			result := planning.Preview(&loaded)
			cmd.Printf(
				"plan ready: team=%s policy=%s days=%d teams=%d policies=%d\n",
				result.TeamID,
				result.PolicyID,
				result.Days,
				result.TeamCount,
				result.PolicyCount,
			)
			return nil
		},
	}

	bindLoadFlags(cmd, &options)
	return cmd
}

func bindLoadFlags(cmd *cobra.Command, options *config.LoadOptions) {
	flags := cmd.Flags()
	flags.StringVar(&options.ConfigPath, "config", "", "Config file path")
	flags.StringVar(&options.TeamOverride, "team", "", "Team ID or config file path")
	flags.StringVar(&options.PolicyOverride, "policy", "", "Policy ID or config file path")
	flags.IntVar(&options.DaysOverride, "days", 0, "Override plan horizon in days")
}
