package domain

import (
	"errors"
	"fmt"
	"time"
)

// Lifecycle defines the factor-specific time boundaries before system limits.
type Lifecycle struct {
	HoldDuration    time.Duration
	IrrelevantAfter time.Duration
}

// SystemLimits bounds lifetimes shared by planning and maintenance.
type SystemLimits struct {
	MaxFactorLifetime time.Duration
}

// ResolvedLifecycle contains the effective lifecycle after system limits apply.
type ResolvedLifecycle struct {
	HoldDuration    time.Duration
	IrrelevantAfter time.Duration
}

// ResolveLifecycle validates a factor lifecycle and applies the system maximum.
func ResolveLifecycle(lifecycle Lifecycle, limits SystemLimits) (ResolvedLifecycle, error) {
	if lifecycle.HoldDuration < 0 {
		return ResolvedLifecycle{}, errors.New("hold duration must not be negative")
	}
	if lifecycle.IrrelevantAfter <= lifecycle.HoldDuration {
		return ResolvedLifecycle{}, fmt.Errorf(
			"irrelevant-after duration %s must be greater than hold duration %s",
			lifecycle.IrrelevantAfter,
			lifecycle.HoldDuration,
		)
	}

	resolved := ResolvedLifecycle{
		HoldDuration:    lifecycle.HoldDuration,
		IrrelevantAfter: lifecycle.IrrelevantAfter,
	}
	if limits.MaxFactorLifetime > 0 && resolved.IrrelevantAfter > limits.MaxFactorLifetime {
		resolved.IrrelevantAfter = limits.MaxFactorLifetime
		if resolved.IrrelevantAfter <= resolved.HoldDuration {
			return ResolvedLifecycle{}, errors.New("maximum factor lifetime must be greater than hold duration")
		}
	}
	return resolved, nil
}
