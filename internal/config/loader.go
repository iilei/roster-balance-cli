package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Load returns the effective config after defaults, discovery, and overrides.
func Load(options LoadOptions) (Config, error) {
	loaded := DefaultConfig()
	v := viper.New()
	setDefaults(v, loaded)

	configPath, explicit, err := discoverConfigPath(options.ConfigPath)
	if err != nil {
		return Config{}, err
	}
	if configPath != "" {
		if err := readConfigFile(v, configPath); err != nil {
			if explicit || !isConfigNotFound(err) {
				return Config{}, err
			}
		}
	}

	if err := v.Unmarshal(&loaded); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	applyOverrides(&loaded, options)
	if err := loaded.Validate(); err != nil {
		return Config{}, err
	}
	return loaded, nil
}

func setDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("plan.options.team.id", cfg.Plan.Options.Team.ID)
	v.SetDefault("plan.options.policy.id", cfg.Plan.Options.Policy.ID)
	v.SetDefault("plan.options.days", cfg.Plan.Options.Days)
	v.SetDefault("teams", cfg.Teams)
	v.SetDefault("policies", cfg.Policies)
	v.SetDefault("factors", cfg.Factors)
}

func discoverConfigPath(configPath string) (string, bool, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return "", true, fmt.Errorf("open config %q: %w", configPath, err)
		}
		return configPath, true, nil
	}

	candidates := []string{".rosterbalance", ".rosterbalance.toml"}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, false, nil
		}
	}
	return "", false, nil
}

func readConfigFile(v *viper.Viper, path string) error {
	if filepath.Base(path) == ".rosterbalance" {
		v.SetConfigType("toml")
	}
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

func applyOverrides(cfg *Config, options LoadOptions) {
	if options.TeamOverride != "" {
		cfg.Plan.Options.Team.ID = options.TeamOverride
	}
	if options.PolicyOverride != "" {
		cfg.Plan.Options.Policy.ID = options.PolicyOverride
	}
	if options.DaysOverride > 0 {
		cfg.Plan.Options.Days = options.DaysOverride
	}
}

// Validate validates the effective config against schema and semantic rules.
func (cfg Config) Validate() error {
	if err := validateSchema(cfg); err != nil {
		return err
	}
	if cfg.Plan.Options.Team.ID == "" {
		return errors.New("plan.options.team.id is required")
	}
	if cfg.Plan.Options.Policy.ID == "" {
		return errors.New("plan.options.policy.id is required")
	}
	if cfg.Plan.Options.Days <= 0 {
		return errors.New("plan.options.days must be greater than zero")
	}
	return nil
}

func normalizeJSON(cfg Config) ([]byte, error) {
	return json.Marshal(cfg)
}

func isConfigNotFound(err error) bool {
	var notFound viper.ConfigFileNotFoundError
	if errors.As(err, &notFound) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "file not found")
}
