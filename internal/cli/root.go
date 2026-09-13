package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/planning"
	"github.com/spf13/cobra"
)

// Version contains the build metadata injected at build time.
type Version struct {
	Version string
	Commit  string
	Date    string
}

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
	var decayProfileOverride string

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
				decayProfile := factor.Lifecycle.DecayProfile
				if decayProfileOverride != "" {
					decayProfile = decayProfileOverride
				}

				hold, err := parseLifecycleDuration(factor.Lifecycle.HoldDuration)
				if err != nil {
					return fmt.Errorf("factor %q hold_duration: %w", factor.Name, err)
				}
				irrelevantAfter, err := parseLifecycleDuration(factor.Lifecycle.IrrelevantAfter)
				if err != nil {
					return fmt.Errorf("factor %q irrelevant_after: %w", factor.Name, err)
				}
				curve, err := buildDecayCurve(decayProfile)
				if err != nil {
					return fmt.Errorf("factor %q: %w", factor.Name, err)
				}
				output.Factors = append(output.Factors, factorOutputItem{
					Name: factor.Name,
					Lifecycle: lifecycleOutput{
						DecayProfile:         decayProfile,
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
	cmd.Flags().StringVar(&decayProfileOverride, "decay-profile", "", "Override the decay profile in inspection output")
	return cmd
}

type factorOutput struct {
	SchemaVersion string             `json:"schema_version"`
	Factors       []factorOutputItem `json:"factors"`
}

type factorOutputItem struct {
	Name      string          `json:"name"`
	Lifecycle lifecycleOutput `json:"lifecycle"`
}

type lifecycleOutput struct {
	DecayProfile         string           `json:"decay_profile"`
	HoldDurationHours    float64          `json:"hold_duration_hours"`
	IrrelevantAfterHours float64          `json:"irrelevant_after_hours"`
	Curve                decayCurveOutput `json:"curve"`
}

type decayCurveOutput struct {
	Kind     string       `json:"kind"`
	Exponent float64      `json:"exponent,omitempty"`
	Samples  []curvePoint `json:"samples"`
}

type curvePoint struct {
	X      float64 `json:"x"`
	Impact float64 `json:"impact"`
}

func buildDecayCurve(profile string) (decayCurveOutput, error) {
	const sampleCount = 16

	curve := decayCurveOutput{
		Kind:    "power",
		Samples: make([]curvePoint, 0, sampleCount+1),
	}
	switch profile {
	case "hard-drop":
		curve.Kind = "step"
	case "front-loaded":
		curve.Exponent = 0.1
	case "linear":
		curve.Exponent = 1
	case "back-loaded":
		curve.Exponent = 4
	case "flat":
		curve.Kind = "step"
	default:
		return decayCurveOutput{}, fmt.Errorf("unsupported decay profile %q", profile)
	}

	for index := 0; index <= sampleCount; index++ {
		x := float64(index) / sampleCount
		impact := 0.0
		switch profile {
		case "hard-drop":
			if index == 0 {
				impact = 1
			}
		case "flat":
			if index < sampleCount {
				impact = 1
			}
		default:
			impact = 1 - math.Pow(x, curve.Exponent)
		}
		curve.Samples = append(curve.Samples, curvePoint{X: x, Impact: impact})
	}
	return curve, nil
}

func parseLifecycleDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(value, "d"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days * float64(24*time.Hour)), nil
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
				return fmt.Errorf("validate does not accept positional arguments")
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
				return fmt.Errorf("plan does not accept positional arguments")
			}

			loaded, err := config.Load(options)
			if err != nil {
				return err
			}

			result := planning.Preview(loaded)
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
