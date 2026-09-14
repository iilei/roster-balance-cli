package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	dataio "github.com/iilei/roster-balance-cli/internal/io"
	"github.com/iilei/roster-balance-cli/internal/planning"
	"github.com/spf13/cobra"
)

//nolint:govet // The output preserves the planning instant as an RFC 3339 timestamp.
type recommendationInspectionOutput struct {
	Candidates    []planning.CandidateEvidence `json:"candidates"`
	SchemaVersion string                       `json:"schema_version"`
	Duty          string                       `json:"duty"`
	Now           time.Time                    `json:"now"`
	LookbackHours float64                      `json:"lookback_hours"`
}

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
				Candidates:    evidence.Candidates,
			}
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
