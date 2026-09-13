package config

const (
	defaultTeamID   = "5c05fb0b-416a-4d75-9991-94c13c418ec4"
	defaultPolicyID = "7659cd1f-651e-4d76-ae9e-6ba98286233c"
	defaultDays     = 7
)

// DefaultConfig returns the built-in config values.
func DefaultConfig() Config {
	return Config{
		Plan: PlanConfig{
			Options: PlanOptions{
				Team:   EntityRef{ID: defaultTeamID},
				Policy: EntityRef{ID: defaultPolicyID},
				Days:   defaultDays,
			},
		},
		Teams:    []Team{{ID: defaultTeamID}},
		Policies: []Policy{{ID: defaultPolicyID}},
	}
}
