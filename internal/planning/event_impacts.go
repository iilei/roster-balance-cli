package planning

import (
	"errors"
	"fmt"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/domain"
)

const rosterPenaltyEffect = "roster-penalty"

// ResolveEventEffects applies an occurrence's configured impacts as planning effects.
func ResolveEventEffects(cfg *config.Config, event *domain.EventOccurrence) ([]domain.Effect, error) {
	eventType, found := cfg.EventTypes[event.Type]
	if !found {
		return nil, nil
	}

	maximum, err := config.ParseElapsedDuration(cfg.Limits.MaxIrrelevantAfter)
	if err != nil {
		return nil, fmt.Errorf("max_irrelevant_after: %w", err)
	}
	if maximum <= 0 {
		return nil, errors.New("max_irrelevant_after must be greater than zero")
	}

	effects := make([]domain.Effect, 0, len(eventType.Impacts))
	for _, application := range eventType.Impacts {
		effect, err := resolveImpactApplication(cfg, event, application, maximum)
		if err != nil {
			return nil, err
		}
		effects = append(effects, effect)
	}
	return effects, nil
}

func resolveImpactApplication(
	cfg *config.Config,
	event *domain.EventOccurrence,
	application config.ImpactApplication,
	maximum time.Duration,
) (domain.Effect, error) {
	impactMath, found := cfg.ImpactMaths[application.ImpactMath]
	if !found {
		return domain.Effect{}, fmt.Errorf(
			"event type %q references unknown impact math %q",
			event.Type,
			application.ImpactMath,
		)
	}
	startsAt, err := impactStart(event, impactMath.StartsFrom)
	if err != nil {
		return domain.Effect{}, err
	}
	hold, err := config.ParseElapsedDuration(impactMath.Lifecycle.HoldDuration)
	if err != nil {
		return domain.Effect{}, fmt.Errorf("impact math %q hold_duration: %w", application.ImpactMath, err)
	}
	irrelevantAfter, err := config.ParseElapsedDuration(impactMath.Lifecycle.IrrelevantAfter)
	if err != nil {
		return domain.Effect{}, fmt.Errorf("impact math %q irrelevant_after: %w", application.ImpactMath, err)
	}
	lifecycle, err := domain.ResolveLifecycle(
		domain.Lifecycle{HoldDuration: hold, IrrelevantAfter: irrelevantAfter},
		domain.SystemLimits{MaxFactorLifetime: maximum},
	)
	if err != nil {
		return domain.Effect{}, fmt.Errorf("impact math %q: %w", application.ImpactMath, err)
	}
	if application.Effect != rosterPenaltyEffect {
		return domain.Effect{}, fmt.Errorf("unsupported impact effect %q", application.Effect)
	}
	return domain.RosterPenalty(
		event,
		application.Role,
		startsAt,
		lifecycle.IrrelevantAfter,
		application.ImpactMath,
	), nil
}

func impactStart(event *domain.EventOccurrence, startsFrom string) (time.Time, error) {
	switch startsFrom {
	case "event.starts_at":
		return event.StartsAt, nil
	case "event.ends_at":
		return event.EndsAt(), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported impact starts_from %q", startsFrom)
	}
}
