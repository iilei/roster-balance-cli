package config

// Config describes the effective rosterbalance configuration.
type Config struct {
	Plan     PlanConfig `json:"plan"               mapstructure:"plan"`
	Teams    []Team     `json:"teams,omitempty"    mapstructure:"teams"`
	Policies []Policy   `json:"policies,omitempty" mapstructure:"policies"`
}

// PlanConfig holds planning defaults and options.
type PlanConfig struct {
	Options PlanOptions `json:"options" mapstructure:"options"`
}

// PlanOptions holds the selected team, policy, and planning horizon.
type PlanOptions struct {
	Team   EntityRef `json:"team"   mapstructure:"team"`
	Policy EntityRef `json:"policy" mapstructure:"policy"`
	Days   int       `json:"days"   mapstructure:"days"`
}

// EntityRef identifies a team or policy.
type EntityRef struct {
	ID string `json:"id" mapstructure:"id"`
}

// Team represents a roster team definition.
type Team struct {
	ID string `json:"id" mapstructure:"id"`
}

// Policy represents a roster policy definition.
type Policy struct {
	ID string `json:"id" mapstructure:"id"`
}

// LoadOptions configures config discovery and overrides.
type LoadOptions struct {
	ConfigPath     string
	TeamOverride   string
	PolicyOverride string
	DaysOverride   int
}
