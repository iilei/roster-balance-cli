package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/iilei/roster-balance-cli/internal/cli"
)

const (
	developmentVersion = "dev"
	noCommit           = "none"
	unknownBuildDate   = "unknown"
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
	root.SetArgs([]string{"inspect", "factors"})
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
