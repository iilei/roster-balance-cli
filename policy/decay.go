// Package policy evaluates roster policy logic.
package policy

import (
	_ "embed"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	starlarkmath "github.com/canonical/starlark/lib/math"
	"github.com/canonical/starlark/starlark"
	"github.com/canonical/starlark/syntax"
)

const (
	decayProfileName     = "DecayProfile"
	evaluatorKeyword     = "evaluator"
	descriptionKeyword   = "description"
	defaultBuiltinSource = "builtin"
)

var (
	defaultDecayRegistry     *DecayRegistry
	defaultDecayRegistryOnce sync.Once
	//go:embed builtins.star
	decayBuiltinsSource []byte
)

// EvaluatorDescriptor describes a decay evaluator's metadata and callable.
type (
	EvaluatorDescriptor struct {
		Evaluator   starlark.Callable
		Reference   string
		Origin      string
		Language    string
		Source      string
		Description string
	}

	// DecayRegistry stores decay profile evaluators and metadata.
	DecayRegistry struct {
		values map[string]EvaluatorDescriptor
		mu     sync.RWMutex
	}

	decayProfileValue struct {
		evaluator   starlark.Callable
		description string
	}
)

// DefaultDecayRegistry returns the package default built-in decay registry.
func DefaultDecayRegistry() *DecayRegistry {
	defaultDecayRegistryOnce.Do(func() {
		rect := NewDecayRegistry()
		if err := rect.LoadBuiltins(); err != nil {
			panic(fmt.Sprintf("load built-in decay profiles: %v", err))
		}
		defaultDecayRegistry = rect
	})
	return defaultDecayRegistry
}

// NewDecayRegistry creates a fresh decay registry without any built-ins.
func NewDecayRegistry() *DecayRegistry {
	return &DecayRegistry{values: make(map[string]EvaluatorDescriptor)}
}

// LoadBuiltins registers the embedded Starlark decay profiles.
func (r *DecayRegistry) LoadBuiltins() error {
	if len(decayBuiltinsSource) == 0 {
		return errors.New("built-in decay source is empty")
	}
	thread := &starlark.Thread{Name: "rosterbalance-decay"}
	_, err := starlark.ExecFileOptions(
		&syntax.FileOptions{},
		thread,
		"builtins.star",
		string(decayBuiltinsSource),
		starlark.StringDict{
			decayProfileName: starlark.NewBuiltin(decayProfileName, decayProfile),
			"math":           starlarkmath.Module,
			"register": starlark.NewBuiltin("register", func(
				_ *starlark.Thread,
				_ *starlark.Builtin,
				args starlark.Tuple,
				kwargs []starlark.Tuple,
			) (starlark.Value, error) {
				if len(args) != 0 {
					return nil, errors.New("register accepts keyword arguments only")
				}
				var alias string
				var evaluator starlark.Value
				for _, kwarg := range kwargs {
					key, ok := kwarg[0].(starlark.String)
					if !ok {
						return nil, errors.New("register keyword name is not a string")
					}
					switch string(key) {
					case "alias":
						value, err := starlarkString(kwarg[1])
						if err != nil {
							return nil, fmt.Errorf("register alias: %w", err)
						}
						alias = value
					case evaluatorKeyword:
						evaluator = kwarg[1]
					}
				}
				if alias == "" {
					return nil, errors.New("register requires alias")
				}
				callable, ok := evaluator.(starlark.Callable)
				if !ok {
					return nil, errors.New("register expects a callable evaluator")
				}
				if err := r.Register(alias, callable, "builtins.star"); err != nil {
					return nil, err
				}
				return starlark.None, nil
			}),
		},
	)
	return err
}

// Register stores a decay evaluator under an alias.
func (r *DecayRegistry) Register(alias string, evaluator starlark.Callable, source string) error {
	if r == nil {
		return errors.New("decay registry is nil")
	}
	if evaluator == nil {
		return errors.New("decay evaluator is nil")
	}
	canonical := normalizeAlias(alias)
	if canonical == "" {
		return errors.New("decay alias must not be empty")
	}
	if source == "" {
		source = defaultBuiltinSource
	}
	descText := descriptionFor(evaluator)
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.values[canonical]; exists {
		return fmt.Errorf("decay profile %q is already registered", canonical)
	}
	r.values[canonical] = EvaluatorDescriptor{
		Reference:   canonical,
		Origin:      "builtin",
		Language:    "starlark",
		Source:      source,
		Description: descText,
		Evaluator:   evaluator,
	}
	return nil
}

// Describe returns evaluator metadata for an alias, if present.
func (r *DecayRegistry) Describe(alias string) (EvaluatorDescriptor, bool) {
	if r == nil {
		return EvaluatorDescriptor{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, ok := r.values[normalizeAlias(alias)]
	return profile, ok
}

// Evaluate computes the decay impact for a normalized lifecycle point x in [0, 1].
func (r *DecayRegistry) Evaluate(alias string, x float64) (float64, error) {
	if r == nil {
		return 0, errors.New("decay registry is nil")
	}
	if x < 0 || x > 1 {
		return 0, fmt.Errorf("decay x = %g, want value in [0, 1]", x)
	}
	desc, ok := r.Describe(alias)
	if !ok {
		return 0, fmt.Errorf("unsupported decay profile %q", alias)
	}
	thread := &starlark.Thread{Name: "rosterbalance-decay-eval"}
	result, err := starlark.Call(thread, desc.Evaluator, starlark.Tuple{starlark.Float(x)}, nil)
	if err != nil {
		return 0, fmt.Errorf("evaluate %q: %w", alias, err)
	}
	impact, ok := starlark.AsFloat(result)
	if !ok {
		return 0, fmt.Errorf("evaluate %q: result is not numeric", alias)
	}
	if math.IsNaN(impact) || math.IsInf(impact, 0) {
		return 0, fmt.Errorf("evaluate %q: result must be finite", alias)
	}
	if impact < 0 || impact > 1 {
		return 0, fmt.Errorf("evaluate %q: result out of range [0, 1]: %g", alias, impact)
	}
	return impact, nil
}

func normalizeAlias(alias string) string {
	return strings.TrimSpace(strings.ToLower(alias))
}

func descriptionFor(value starlark.Callable) string {
	if profile, ok := value.(interface{ Description() string }); ok {
		if text := profile.Description(); text != "" {
			return text
		}
	}
	if hasAttrs, ok := value.(starlark.HasAttrs); ok {
		if attr, err := hasAttrs.Attr(descriptionKeyword); err == nil {
			if text, ok := starlarkString(attr); ok == nil {
				return text
			}
		}
	}
	if text := value.String(); text != "" {
		return text
	}
	return "<callable>"
}

func (d decayProfileValue) Attr(name string) (starlark.Value, error) {
	switch name {
	case evaluatorKeyword:
		return d.evaluator, nil
	case descriptionKeyword:
		return starlark.String(d.description), nil
	default:
		return nil, starlark.NoSuchAttrError(fmt.Sprintf("DecayProfile has no attribute %q", name))
	}
}

func (d decayProfileValue) AttrNames() []string {
	return []string{evaluatorKeyword, descriptionKeyword}
}

func (d decayProfileValue) Name() string { return decayProfileName }

func (d decayProfileValue) CallInternal(
	thread *starlark.Thread,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	return starlark.Call(thread, d.evaluator, args, kwargs)
}

func (d decayProfileValue) String() string {
	if d.description != "" {
		return d.description
	}
	return d.evaluator.String()
}

func (d decayProfileValue) Type() string { return decayProfileName }

func (d decayProfileValue) Freeze() {}

func (d decayProfileValue) Truth() starlark.Bool { return starlark.True }

func (d decayProfileValue) Hash() (uint32, error) { return 0, nil }

func (d decayProfileValue) Description() string { return d.description }

func decayProfile(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("%s expects at most one callable argument", decayProfileName)
	}
	var description string
	for _, kwarg := range kwargs {
		key, ok := kwarg[0].(starlark.String)
		if !ok {
			return nil, errors.New("DecayProfile keyword name is not a string")
		}
		switch string(key) {
		case descriptionKeyword:
			value, err := starlarkString(kwarg[1])
			if err != nil {
				return nil, fmt.Errorf("%s description: %w", decayProfileName, err)
			}
			description = value
		default:
			return nil, fmt.Errorf("%s does not accept keyword %q", decayProfileName, key)
		}
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("%s expects one callable argument", decayProfileName)
	}
	callable, ok := args[0].(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("%s expects a callable argument", decayProfileName)
	}
	if description == "" {
		description = callable.String()
	}
	return decayProfileValue{evaluator: callable, description: description}, nil
}
