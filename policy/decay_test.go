package policy

import (
	"strings"
	"testing"

	"github.com/canonical/starlark/starlark"
)

type stubCallable struct{}

func (stubCallable) String() string { return "stub" }
func (stubCallable) Type() string   { return "stub" }
func (stubCallable) Freeze()        {}
func (stubCallable) Truth() starlark.Bool {
	return starlark.True
}
func (stubCallable) Hash() (uint32, error) { return 0, nil }
func (stubCallable) Name() string          { return "stub" }
func (stubCallable) CallInternal(
	_ *starlark.Thread,
	_ starlark.Tuple,
	_ []starlark.Tuple,
) (starlark.Value, error) {
	return starlark.Float(1), nil
}

func TestDefaultDecayRegistryIncludesBuiltins(t *testing.T) {
	reg := DefaultDecayRegistry()
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
	reg := NewDecayRegistry()
	if err := reg.Register("linear", stubCallable{}, "stub"); err != nil {
		t.Fatalf("Register() first call error = %v", err)
	}
	if err := reg.Register("linear", stubCallable{}, "stub"); err == nil {
		t.Fatal("Register() duplicate alias unexpectedly succeeded")
	}
}

func TestDecayProfileDescriptionIsAttachedToWrapper(t *testing.T) {
	profile := decayProfileValue{
		evaluator:   stubCallable{},
		description: "lambda x: 1.0 - math.pow(x, 4.0)",
	}
	if got := profile.Description(); !strings.Contains(got, "math.pow") {
		t.Fatalf("profile.Description() = %q, want description containing math.pow", got)
	}
	if got := profile.String(); !strings.Contains(got, "math.pow") {
		t.Fatalf("profile.String() = %q, want description containing math.pow", got)
	}
}
