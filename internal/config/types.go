package config

type (
	// Config describes the effective rosterbalance configuration.
	Config struct {
		Limits      SystemLimits          `json:"limits"                 mapstructure:"limits"`
		Teams       []Team                `json:"teams,omitempty"        mapstructure:"teams"`
		Policies    []Policy              `json:"policies,omitempty"     mapstructure:"policies"`
		Factors     []Factor              `json:"factors,omitempty"      mapstructure:"factors"`
		ImpactMaths map[string]ImpactMath `json:"impact_maths,omitempty" mapstructure:"impact_maths"`
		EventTypes  map[string]EventType  `json:"event_types,omitempty"  mapstructure:"event_types"`
		Plan        PlanConfig            `json:"plan"                   mapstructure:"plan"`
	}

	// SystemLimits bounds lifecycle durations across planning inputs.
	SystemLimits struct {
		MaxIrrelevantAfter string `json:"max_irrelevant_after" mapstructure:"max_irrelevant_after"`
	}

	// PlanConfig holds planning defaults and options.
	PlanConfig struct {
		Options PlanOptions `json:"options" mapstructure:"options"`
	}

	// PlanOptions holds the selected team, policy, and planning horizon.
	PlanOptions struct {
		Team   EntityRef `json:"team"   mapstructure:"team"`
		Policy EntityRef `json:"policy" mapstructure:"policy"`
		Days   int       `json:"days"   mapstructure:"days"`
	}

	// EntityRef identifies a team or policy.
	EntityRef struct {
		ID string `json:"id" mapstructure:"id"`
	}

	// Team represents a roster team definition.
	Team struct {
		ID                         string  `json:"id"                            mapstructure:"id"`
		RosterPenaltyLockThreshold float64 `json:"roster_penalty_lock_threshold" mapstructure:"roster_penalty_lock_threshold"`
	}

	// Policy represents a roster policy definition.
	Policy struct {
		ID string `json:"id" mapstructure:"id"`
	}

	// Factor describes a planning factor and its lifecycle configuration.
	Factor struct {
		Name      string    `json:"name"      mapstructure:"name"`
		Lifecycle Lifecycle `json:"lifecycle" mapstructure:"lifecycle"`
	}

	// Lifecycle describes how a factor's impact changes over time.
	Lifecycle struct {
		DecayProfile    string `json:"decay_profile"    mapstructure:"decay_profile"`
		HoldDuration    string `json:"hold_duration"    mapstructure:"hold_duration"`
		IrrelevantAfter string `json:"irrelevant_after" mapstructure:"irrelevant_after"`
	}

	// ImpactMath defines a reusable impact lifecycle anchored to an occurrence boundary.
	ImpactMath struct {
		StartsFrom string    `json:"starts_from" mapstructure:"starts_from"`
		Lifecycle  Lifecycle `json:"lifecycle"   mapstructure:"lifecycle"`
	}

	// EventType composes the impacts produced by an occurrence type.
	EventType struct {
		Impacts []ImpactApplication `json:"impacts" mapstructure:"impacts"`
	}

	// ImpactApplication applies reusable impact math as a planning effect.
	ImpactApplication struct {
		ImpactMath string `json:"impact_math" mapstructure:"impact_math"`
		Effect     string `json:"effect"      mapstructure:"effect"`
		Role       string `json:"role"        mapstructure:"role"`
	}

	// LoadOptions configures config discovery and overrides.
	LoadOptions struct {
		ConfigPath     string
		TeamOverride   string
		PolicyOverride string
		DaysOverride   int
	}
)
