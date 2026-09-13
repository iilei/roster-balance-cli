package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadUsesBuiltInDefaultsWhenNoConfigIsPresent(t *testing.T) {
	loaded, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := DefaultConfig()
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

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	loaded, err := Load(LoadOptions{DaysOverride: 21})
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
