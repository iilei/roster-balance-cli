package planning

import "github.com/iilei/roster-balance-cli/internal/config"

// PreviewResult summarizes the effective planning inputs.
type PreviewResult struct {
	TeamID      string
	PolicyID    string
	Days        int
	TeamCount   int
	PolicyCount int
}

// Preview returns a lightweight planning summary for the current config.
func Preview(cfg config.Config) PreviewResult {
	return PreviewResult{
		TeamID:      cfg.Plan.Options.Team.ID,
		PolicyID:    cfg.Plan.Options.Policy.ID,
		Days:        cfg.Plan.Options.Days,
		TeamCount:   len(cfg.Teams),
		PolicyCount: len(cfg.Policies),
	}
}
