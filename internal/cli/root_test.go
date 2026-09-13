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
				Curve                struct {
					Kind    string `json:"kind"`
					Samples []struct {
						X      float64 `json:"x"`
						Impact float64 `json:"impact"`
					} `json:"samples"`
				} `json:"curve"`
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
	if got.Factors[0].Lifecycle.Curve.Kind != "power" {
		t.Fatalf("curve kind = %q", got.Factors[0].Lifecycle.Curve.Kind)
	}
	if len(got.Factors[0].Lifecycle.Curve.Samples) != 49 {
		t.Fatalf("curve sample count = %d, want 49", len(got.Factors[0].Lifecycle.Curve.Samples))
	}
	if got.Factors[0].Lifecycle.Curve.Samples[24].Impact >= 0.5 {
		t.Fatalf(
			"front-loaded midpoint impact = %g, want less than 0.5",
			got.Factors[0].Lifecycle.Curve.Samples[24].Impact,
		)
	}
}

func TestDecayCurvesStayCurvedAfterHold(t *testing.T) {
	front, err := buildDecayCurve("front-loaded", 48, 336)
	if err != nil {
		t.Fatalf("front-loaded curve error = %v", err)
	}
	back, err := buildDecayCurve("back-loaded", 48, 336)
	if err != nil {
		t.Fatalf("back-loaded curve error = %v", err)
	}
	if front.Samples[7].Impact <= 0 || front.Samples[7].Impact >= 1 {
		t.Fatalf("front-loaded sample after hold = %v, want a gradual curve value in (0,1)", front.Samples[7].Impact)
	}
	if back.Samples[7].Impact <= 0 || back.Samples[7].Impact >= 1 {
		t.Fatalf("back-loaded sample after hold = %v, want a gradual curve value in (0,1)", back.Samples[7].Impact)
	}
	if front.Samples[7].Impact <= front.Samples[8].Impact {
		t.Fatalf(
			"front-loaded decay should decrease after hold: %v -> %v",
			front.Samples[7].Impact,
			front.Samples[8].Impact,
		)
	}
	if back.Samples[7].Impact <= back.Samples[8].Impact {
		t.Fatalf(
			"back-loaded decay should fall gradually after hold: %v -> %v",
			back.Samples[7].Impact,
			back.Samples[8].Impact,
		)
	}
}
