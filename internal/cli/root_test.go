package cli

import (
	"bytes"
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
