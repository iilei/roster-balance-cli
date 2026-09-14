package io_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	dataio "github.com/iilei/roster-balance-cli/internal/io"
)

func TestLoadTeamMembersReadsJSONArray(t *testing.T) {
	path := filepath.Join(t.TempDir(), "members.json")
	content := []byte(`[{"member_id":"member-1","joined_at":"2025-01-01T00:00:00Z","eligible_duties":["on-call"]}]`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	members, err := dataio.LoadTeamMembers([]string{path})
	if err != nil {
		t.Fatalf("LoadTeamMembers() error = %v", err)
	}
	if len(members) != 1 || !members[0].EligibleForDuty("on-call") {
		t.Fatalf("members = %#v, want duty-eligible member", members)
	}
}

func TestLoadEventOccurrencesReadsJSONLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "occurrences.jsonl")
	content := []byte(
		"{\"id\":\"call-1\",\"type\":\"on-call-call-answered\",\"member_id\":\"member-1\",\"starts_at\":\"2026-09-14T09:15:00Z\",\"duration\":\"27m\"}\n",
	)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	occurrences, err := dataio.LoadEventOccurrences([]string{path})
	if err != nil {
		t.Fatalf("LoadEventOccurrences() error = %v", err)
	}
	if got, want := occurrences[0].EndsAt(), time.Date(2026, 9, 14, 9, 42, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("EndsAt() = %s, want %s", got, want)
	}
}
