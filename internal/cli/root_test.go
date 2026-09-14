package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iilei/roster-balance-cli/internal/cli"
)

const (
	developmentVersion = "dev"
	noCommit           = "none"
	unknownBuildDate   = "unknown"
	inspectCommand     = "inspect"
	inspectFactors     = "factors"
	configFlag         = "--config"
	recommendationNow  = "2026-09-14T10:00:00Z"
)

func TestPlanCommandRunsWithoutArguments(t *testing.T) {
	t.Parallel()
	root := cli.NewRootCommand(cli.Version{Version: developmentVersion, Commit: noCommit, Date: unknownBuildDate})
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"plan"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "plan ready:") {
		t.Fatalf("Execute() output %q does not contain plan result", got)
	}
}

func TestRootHelpDoesNotExposeCompletionCommand(t *testing.T) {
	t.Parallel()
	root := cli.NewRootCommand(cli.Version{Version: developmentVersion, Commit: noCommit, Date: unknownBuildDate})
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Contains(output.String(), "completion") {
		t.Fatalf("help output exposes completion command: %q", output.String())
	}
}

func TestInspectFactorsPrintsDefaultJSON(t *testing.T) {
	t.Parallel()
	root := cli.NewRootCommand(cli.Version{Version: developmentVersion, Commit: noCommit, Date: unknownBuildDate})
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{inspectCommand, inspectFactors})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var got struct {
		SchemaVersion string `json:"schema_version"`
		Factors       []struct {
			Name      string `json:"name"`
			Lifecycle struct {
				Curve struct {
					Reference   string `json:"reference"`
					Description string `json:"description"`
					Samples     []struct {
						X      float64 `json:"x"`
						Impact float64 `json:"impact"`
					} `json:"samples"`
				} `json:"curve"`
				HoldDurationHours    float64 `json:"hold_duration_hours"`
				IrrelevantAfterHours float64 `json:"irrelevant_after_hours"`
			} `json:"lifecycle"`
		} `json:"factors"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON output error = %v; output = %q", err, output.String())
	}
	if got.SchemaVersion != "rosterbalance.factor-output/v1" || len(got.Factors) != 1 {
		t.Fatalf("unexpected factor output: %#v", got)
	}
	if got.Factors[0].Lifecycle.Curve.Reference != "front-loaded" ||
		got.Factors[0].Lifecycle.Curve.Description != "lambda x: 1.0 - math.pow(x, 0.1)" {
		t.Fatalf("unexpected factor lifecycle: %#v", got.Factors[0].Lifecycle)
	}
	if got.Factors[0].Lifecycle.Curve.Samples[24].Impact >= 0.5 {
		t.Fatalf(
			"front-loaded midpoint impact = %g, want less than 0.5",
			got.Factors[0].Lifecycle.Curve.Samples[24].Impact,
		)
	}
}

func TestInspectFactorsUsesHardCutoffForEqualLifecycleBoundaries(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "factor.toml")
	content := []byte(`
[plan.options.team]
id = "team-1"

[plan.options.policy]
id = "policy-1"

[plan.options]
days = 7

[[factors]]
name = "call-recovery"

[factors.lifecycle]
decay_profile = "front-loaded"
hold_duration = "24h"
irrelevant_after = "24h"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	root := cli.NewRootCommand(cli.Version{Version: developmentVersion, Commit: noCommit, Date: unknownBuildDate})
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{inspectCommand, inspectFactors, configFlag, configPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got struct {
		Factors []struct {
			Lifecycle struct {
				Curve struct {
					Samples []struct {
						Impact float64 `json:"impact"`
					} `json:"samples"`
				} `json:"curve"`
			} `json:"lifecycle"`
		} `json:"factors"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON output error = %v; output = %q", err, output.String())
	}
	samples := got.Factors[0].Lifecycle.Curve.Samples
	if samples[len(samples)-2].Impact != 1 || samples[len(samples)-1].Impact != 0 {
		t.Fatalf("cutoff samples = %#v, want 1 before EOL and 0 at EOL", samples[len(samples)-2:])
	}
}

func TestInspectRecommendationReportsCandidateEvidence(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.toml")
	teamDataPath := filepath.Join(directory, "team.json")
	trackedDataPath := filepath.Join(directory, "tracked.jsonl")
	configData := []byte(`
[plan.options.team]
id = "team-1"

[plan.options.policy]
id = "policy-1"

[plan.options]
days = 7

[impact_maths.call_recovery]
starts_from = "event.ends_at"

[impact_maths.call_recovery.lifecycle]
decay_profile = "front-loaded"
hold_duration = "24h"
irrelevant_after = "24h"

[[event_types."on-call-call-answered".impacts]]
impact_math = "call_recovery"
effect = "roster-lock"
role = "on-call"
`)
	teamData := []byte(`[{"member_id":"member-1","joined_at":"2026-09-01T00:00:00Z","eligible_duties":["on-call"]}]`)
	trackedData := []byte(
		"{\"id\":\"call-1\",\"type\":\"on-call-call-answered\",\"member_id\":\"member-1\",\"starts_at\":\"2026-09-14T09:15:00Z\",\"duration\":\"27m\"}\n",
	)
	for path, content := range map[string][]byte{
		configPath:      configData,
		teamDataPath:    teamData,
		trackedDataPath: trackedData,
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	root := cli.NewRootCommand(cli.Version{Version: developmentVersion, Commit: noCommit, Date: unknownBuildDate})
	output := &bytes.Buffer{}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{
		inspectCommand,
		"recommendation",
		configFlag, configPath,
		"--duty", "on-call",
		"--now", recommendationNow,
		"--team-data-fs", teamDataPath,
		"--tracked-data-fs", trackedDataPath,
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var got struct {
		Now        string `json:"now"`
		Candidates []struct {
			MemberID    string `json:"member_id"`
			Occurrences []struct {
				ID string `json:"ID"`
			} `json:"occurrences"`
			ActiveLocks []struct {
				Role string `json:"Role"`
			} `json:"active_locks"`
			ImpactCurves []struct {
				Samples []struct {
					Hours  float64 `json:"hours"`
					Impact float64 `json:"impact"`
				} `json:"samples"`
			} `json:"impact_curves"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON output error = %v; output = %q", err, output.String())
	}
	if got.Now != recommendationNow || len(got.Candidates) != 1 {
		t.Fatalf("unexpected recommendation output: %#v", got)
	}
	if got.Candidates[0].MemberID != "member-1" || len(got.Candidates[0].Occurrences) != 1 ||
		len(got.Candidates[0].ActiveLocks) != 1 || len(got.Candidates[0].ImpactCurves) != 1 {
		t.Fatalf("unexpected candidate evidence: %#v", got.Candidates[0])
	}
	samples := got.Candidates[0].ImpactCurves[0].Samples
	if samples[len(samples)-1].Hours <= 0 || samples[len(samples)-1].Impact != 0 {
		t.Fatalf("curve endpoint = %#v, want zero impact after now", samples[len(samples)-1])
	}
}
