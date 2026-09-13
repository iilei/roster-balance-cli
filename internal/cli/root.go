package cli

import (
	"fmt"

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
	}

	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built at: %s)", version.Version, version.Commit, version.Date)
	rootCmd.AddCommand(newValidateCommand())
	rootCmd.AddCommand(newPlanCommand())

	return rootCmd
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
