package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/iilei/roster-balance-cli/internal/config"
)

func TestLoadUsesBuiltInDefaultsWhenNoConfigIsPresent(t *testing.T) {
	loaded, err := config.Load(config.LoadOptions{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := config.DefaultConfig()
	if !reflect.DeepEqual(want, loaded) {
		t.Fatalf("Load() mismatch: want %#v, got %#v", want, loaded)
	}
}

func TestLoadReadsDiscoveredTomlConfigAndAppliesOverrides(t *testing.T) {
	workDir := t.TempDir()
	configPath := filepath.Join(workDir, ".rosterbalance")
	content := []byte(`
[plan.options.team]
id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

[plan.options.policy]
id = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

[plan.options]
days = 14

[[teams]]
id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

[[policies]]
id = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Chdir(workDir)

	loaded, err := config.Load(config.LoadOptions{DaysOverride: 21})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := loaded.Plan.Options.Team.ID, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"; got != want {
		t.Fatalf("team ID = %q, want %q", got, want)
	}
	if got, want := loaded.Plan.Options.Policy.ID, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"; got != want {
		t.Fatalf("policy ID = %q, want %q", got, want)
	}
	if got, want := loaded.Plan.Options.Days, 21; got != want {
		t.Fatalf("days = %d, want %d", got, want)
	}
}
