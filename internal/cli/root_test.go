package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPlanCommandRunsWithoutArguments(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(Version{Version: "dev", Commit: "none", Date: "unknown"})
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

	root := NewRootCommand(Version{Version: "dev", Commit: "none", Date: "unknown"})
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

	root := NewRootCommand(Version{Version: "dev", Commit: "none", Date: "unknown"})
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
				DecayProfile         string  `json:"decay_profile"`
				HoldDurationHours    float64 `json:"hold_duration_hours"`
				IrrelevantAfterHours float64 `json:"irrelevant_after_hours"`
			} `json:"lifecycle"`
		} `json:"factors"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON output error = %v; output = %q", err, output.String())
	}
	if got.SchemaVersion != "rosterbalance.factor-output/v1" {
		t.Fatalf("schema_version = %q", got.SchemaVersion)
	}
	if len(got.Factors) != 1 {
		t.Fatalf("factor count = %d, want 1", len(got.Factors))
	}
	if got.Factors[0].Name != "duty-work-served" {
		t.Fatalf("factor name = %q", got.Factors[0].Name)
	}
	if got.Factors[0].Lifecycle.DecayProfile != "front-loaded" {
		t.Fatalf("decay profile = %q", got.Factors[0].Lifecycle.DecayProfile)
	}
	if got.Factors[0].Lifecycle.HoldDurationHours != 48 {
		t.Fatalf("hold duration = %g", got.Factors[0].Lifecycle.HoldDurationHours)
	}
	if got.Factors[0].Lifecycle.IrrelevantAfterHours != 336 {
		t.Fatalf("irrelevant after = %g", got.Factors[0].Lifecycle.IrrelevantAfterHours)
	}
}
