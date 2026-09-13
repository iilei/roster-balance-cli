package policy_test

import (
	"strings"
	"testing"

	"github.com/canonical/starlark/starlark"
	"github.com/iilei/roster-balance-cli/policy"
)

const stubCallableName = "stub"

type stubCallable struct{}

func (stubCallable) String() string { return stubCallableName }
func (stubCallable) Type() string   { return stubCallableName }
func (stubCallable) Freeze()        {}
func (stubCallable) Truth() starlark.Bool {
	return starlark.True
}
func (stubCallable) Hash() (uint32, error) { return 0, nil }
func (stubCallable) Name() string          { return stubCallableName }
func (stubCallable) CallInternal(
	_ *starlark.Thread,
	_ starlark.Tuple,
	_ []starlark.Tuple,
) (starlark.Value, error) {
	return starlark.Float(1), nil
}

func TestDefaultDecayRegistryIncludesBuiltins(t *testing.T) {
	reg := policy.DefaultDecayRegistry()
	profile, ok := reg.Describe("front-loaded")
	if !ok {
		t.Fatal("front-loaded profile missing from default registry")
	}
	if profile.Origin != "builtin" || profile.Language != "starlark" || profile.Reference != "front-loaded" {
		t.Fatalf("profile metadata = %#v", profile)
	}
	if got, err := reg.Evaluate("front-loaded", 0.5); err != nil || got >= 0.5 {
		t.Fatalf("front-loaded(0.5) = (%v, %v), want less than 0.5", got, err)
	}
	if got, err := reg.Evaluate("back-loaded", 0.5); err != nil || got <= 0.5 {
		t.Fatalf("back-loaded(0.5) = (%v, %v), want greater than 0.5", got, err)
	}
	if got, err := reg.Evaluate("flat", 0.5); err != nil || got != 1.0 {
		t.Fatalf("flat(0.5) = (%v, %v), want 1.0", got, err)
	}
}

func TestDecayRegistryRejectsDuplicateAliases(t *testing.T) {
	reg := policy.NewDecayRegistry()
	if err := reg.Register("linear", stubCallable{}, "stub"); err != nil {
		t.Fatalf("Register() first call error = %v", err)
	}
	if err := reg.Register("linear", stubCallable{}, "stub"); err == nil {
		t.Fatal("Register() duplicate alias unexpectedly succeeded")
	}
}

func TestDecayProfileDescriptionIsExposed(t *testing.T) {
	profile, ok := policy.DefaultDecayRegistry().Describe("back-loaded")
	if !ok {
		t.Fatal("back-loaded profile missing from default registry")
	}
	if got := profile.Description; !strings.Contains(got, "math.pow") {
		t.Fatalf("profile description = %q, want description containing math.pow", got)
	}
}
