package policy

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/canonical/starlark/starlark"
	"github.com/canonical/starlark/starlarkstruct"
	"github.com/canonical/starlark/syntax"
	"github.com/iilei/roster-balance-cli/internal/domain"
)

const maxInstantArgumentCount = 2

// Runtime evaluates event rules and converts their declarative results to domain effects.
type Runtime struct{}

// EvaluateEvent runs on_event(ctx) from a Starlark source file.
func (Runtime) EvaluateEvent(event *domain.EventOccurrence, source string) ([]domain.Effect, error) {
	thread := &starlark.Thread{Name: "rosterbalance-policy"}
	globals, err := starlark.ExecFileOptions(
		&syntax.FileOptions{},
		thread,
		"policy.star",
		source,
		predeclared(thread),
	)
	if err != nil {
		return nil, fmt.Errorf("execute policy: %w", err)
	}
	function, ok := globals["on_event"]
	if !ok {
		return nil, errors.New("policy must define on_event(ctx)")
	}
	result, err := starlark.Call(thread, function, starlark.Tuple{eventContext(event)}, nil)
	if err != nil {
		return nil, fmt.Errorf("evaluate on_event: %w", err)
	}
	return effectsFromValue(event, result)
}

func predeclared(thread *starlark.Thread) starlark.StringDict {
	timeNamespace, _ := starlarkstruct.Make(thread, nil, nil, []starlark.Tuple{
		{starlark.String("max"), starlark.NewBuiltin("max", maxInstant)},
	})
	effectsNamespace, _ := starlarkstruct.Make(thread, nil, nil, []starlark.Tuple{
		{starlark.String("eligibility_lock"), starlark.NewBuiltin("eligibility_lock", eligibilityLock)},
		{starlark.String("factor_event"), starlark.NewBuiltin("factor_event", factorEvent)},
	})
	return starlark.StringDict{
		"time":    timeNamespace,
		"effects": effectsNamespace,
	}
}

func eventContext(event *domain.EventOccurrence) starlark.Value {
	eventFields := []starlark.Tuple{
		{starlark.String("id"), starlark.String(event.ID)},
		{starlark.String("type"), starlark.String(event.Type)},
		{starlark.String("member_id"), starlark.String(event.MemberID)},
		{starlark.String("starts_at"), starlark.MakeInt64(event.StartsAt.Unix())},
		{starlark.String("ends_at"), starlark.MakeInt64(event.EndsAt().Unix())},
	}
	for key, value := range event.Attributes {
		converted, ok := starlarkAttribute(value)
		if ok {
			eventFields = append(eventFields, starlark.Tuple{starlark.String(key), converted})
		}
	}
	eventValue, _ := starlarkstruct.Make(nil, nil, nil, eventFields)
	contextValue, _ := starlarkstruct.Make(nil, nil, nil, []starlark.Tuple{
		{starlark.String("event"), eventValue},
		{starlark.String("subject"), starlark.String(event.MemberID)},
	})
	return contextValue
}

func starlarkAttribute(value any) (starlark.Value, bool) {
	switch value := value.(type) {
	case string:
		return starlark.String(value), true
	case time.Time:
		return starlark.MakeInt64(value.Unix()), true
	case int:
		return starlark.MakeInt(value), true
	case int64:
		return starlark.MakeInt64(value), true
	case float64:
		return starlark.Float(value), true
	case map[string]any:
		fields := make([]starlark.Tuple, 0, len(value))
		for key, nested := range value {
			converted, ok := starlarkAttribute(nested)
			if ok {
				fields = append(fields, starlark.Tuple{starlark.String(key), converted})
			}
		}
		structValue, err := starlarkstruct.Make(nil, nil, nil, fields)
		return structValue, err == nil
	default:
		return nil, false
	}
}

func maxInstant(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	_ []starlark.Tuple,
) (starlark.Value, error) {
	if len(args) != maxInstantArgumentCount {
		return nil, errors.New("time.max expects two instants")
	}
	left, ok := args[0].(starlark.Int)
	if !ok {
		return nil, errors.New("time.max expects integer instants")
	}
	right, ok := args[1].(starlark.Int)
	if !ok {
		return nil, errors.New("time.max expects integer instants")
	}
	leftInt, ok := left.Int64()
	if !ok {
		return nil, errors.New("time.max instant is out of range")
	}
	rightInt, ok := right.Int64()
	if !ok {
		return nil, errors.New("time.max instant is out of range")
	}
	if rightInt > leftInt {
		return right, nil
	}
	return left, nil
}

func eligibilityLock(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	return effectValue("eligibility-lock", args, kwargs)
}

func factorEvent(
	_ *starlark.Thread,
	_ *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (starlark.Value, error) {
	return effectValue("factor-event", args, kwargs)
}

func effectValue(kind string, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("%s accepts keyword arguments only", kind)
	}
	values := map[string]starlark.Value{"kind": starlark.String(kind)}
	for _, kwarg := range kwargs {
		key, ok := kwarg[0].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("%s keyword name is not a string", kind)
		}
		values[string(key)] = kwarg[1]
	}
	result := starlark.NewDict(len(values))
	for key, value := range values {
		if err := result.SetKey(starlark.String(key), value); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func effectsFromValue(event *domain.EventOccurrence, value starlark.Value) ([]domain.Effect, error) {
	list, ok := value.(*starlark.List)
	if !ok {
		return nil, errors.New("on_event must return a list of effects")
	}
	effects := make([]domain.Effect, 0, list.Len())
	for value := range list.Elements() {
		dict, ok := value.(*starlark.Dict)
		if !ok {
			return nil, errors.New("effect must be a dictionary")
		}
		kind, err := dictString(dict, "kind")
		if err != nil {
			return nil, err
		}
		memberID := event.MemberID
		if value, found, _ := dict.Get(starlark.String("member")); found {
			memberID, err = starlarkString(value)
			if err != nil {
				return nil, err
			}
		}
		switch domain.EffectKind(kind) {
		case domain.EffectFactorEvent:
			factor, err := dictString(dict, "factor")
			if err != nil {
				return nil, err
			}
			effects = append(
				effects,
				domain.Effect{
					Kind:          domain.EffectFactorEvent,
					SourceEventID: event.ID,
					MemberID:      memberID,
					Factor:        factor,
					StartsAt:      event.StartsAt,
				},
			)
		case domain.EffectEligibilityLock:
			role, err := dictString(dict, "role")
			if err != nil {
				return nil, err
			}
			startsAt, err := dictTime(dict, "starts_at", event.StartsAt)
			if err != nil {
				return nil, err
			}
			durationHours, err := dictNumber(dict, "duration_hours")
			if err != nil {
				return nil, err
			}
			reason, _ := optionalDictString(dict, "reason")
			effects = append(
				effects,
				domain.EligibilityLock(event, role, startsAt, time.Duration(durationHours*float64(time.Hour)), reason),
			)
		default:
			return nil, fmt.Errorf("unsupported effect kind %q", kind)
		}
	}
	return effects, nil
}

func dictString(dict *starlark.Dict, key string) (string, error) {
	value, found, err := dict.Get(starlark.String(key))
	if err != nil || !found {
		return "", fmt.Errorf("effect requires %q", key)
	}
	return starlarkString(value)
}

func optionalDictString(dict *starlark.Dict, key string) (string, error) {
	value, found, err := dict.Get(starlark.String(key))
	if err != nil || !found {
		return "", err
	}
	return starlarkString(value)
}

func dictTime(dict *starlark.Dict, key string, fallback time.Time) (time.Time, error) {
	value, found, err := dict.Get(starlark.String(key))
	if err != nil {
		return time.Time{}, err
	}
	if !found {
		return fallback, nil
	}
	seconds, err := starlarkNumber(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("effect %q: %w", key, err)
	}
	return time.Unix(int64(seconds), int64((seconds-math.Trunc(seconds))*float64(time.Second))).UTC(), nil
}

func dictNumber(dict *starlark.Dict, key string) (float64, error) {
	value, found, err := dict.Get(starlark.String(key))
	if err != nil || !found {
		return 0, fmt.Errorf("effect requires %q", key)
	}
	return starlarkNumber(value)
}

func starlarkNumber(value starlark.Value) (float64, error) {
	switch value := value.(type) {
	case starlark.Int:
		integer, ok := value.Int64()
		if !ok {
			return 0, errors.New("integer is out of range")
		}
		return float64(integer), nil
	case starlark.Float:
		return float64(value), nil
	default:
		return 0, fmt.Errorf("expected number, got %s", value.Type())
	}
}

func starlarkString(value starlark.Value) (string, error) {
	stringValue, ok := value.(starlark.String)
	if !ok {
		return "", fmt.Errorf("expected string, got %s", value.Type())
	}
	return string(stringValue), nil
}
