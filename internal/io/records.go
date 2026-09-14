// Package io loads externally managed roster data.
package io

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iilei/roster-balance-cli/internal/config"
	"github.com/iilei/roster-balance-cli/internal/domain"
)

type (
	// TeamMember is the externally supplied team and duty-eligibility projection.
	TeamMember struct {
		MemberID       string    `json:"member_id"`
		JoinedAt       time.Time `json:"joined_at"`
		EligibleDuties []string  `json:"eligible_duties"`
	}

	//nolint:govet // RFC 3339 timestamps make the external record format explicit.
	occurrenceRecord struct {
		Attributes map[string]any `json:"attributes"`
		ID         string         `json:"id"`
		Type       string         `json:"type"`
		MemberID   string         `json:"member_id"`
		Duration   string         `json:"duration"`
		StartsAt   time.Time      `json:"starts_at"`
	}
)

// LoadTeamMembers reads JSON arrays or JSON Lines from externally managed files.
func LoadTeamMembers(paths []string) ([]TeamMember, error) {
	members := make([]TeamMember, 0)
	for _, path := range paths {
		var records []TeamMember
		if err := decodeRecords(path, &records); err != nil {
			return nil, err
		}
		for index := range records {
			member := &records[index]
			if member.MemberID == "" {
				return nil, recordError(path, index, errors.New("member_id is required"))
			}
			if member.JoinedAt.IsZero() {
				return nil, recordError(path, index, errors.New("joined_at is required"))
			}
			members = append(members, *member)
		}
	}
	return members, nil
}

// LoadEventOccurrences reads JSON arrays or JSON Lines from externally managed files.
func LoadEventOccurrences(paths []string) ([]domain.EventOccurrence, error) {
	occurrences := make([]domain.EventOccurrence, 0)
	ids := make(map[string]struct{})
	for _, path := range paths {
		var records []occurrenceRecord
		if err := decodeRecords(path, &records); err != nil {
			return nil, err
		}
		for index := range records {
			record := &records[index]
			duration, err := config.ParseElapsedDuration(record.Duration)
			if err != nil {
				return nil, recordError(path, index, fmt.Errorf("duration: %w", err))
			}
			occurrence := domain.EventOccurrence{
				ID:         record.ID,
				Type:       record.Type,
				MemberID:   record.MemberID,
				StartsAt:   record.StartsAt,
				Duration:   duration,
				Attributes: record.Attributes,
			}
			if err := validateOccurrence(&occurrence); err != nil {
				return nil, recordError(path, index, err)
			}
			if _, exists := ids[occurrence.ID]; exists {
				return nil, recordError(path, index, fmt.Errorf("duplicate occurrence ID %q", occurrence.ID))
			}
			ids[occurrence.ID] = struct{}{}
			occurrences = append(occurrences, occurrence)
		}
	}
	return occurrences, nil
}

// #nosec G304 -- paths are explicit CLI arguments from the invoking user.
func decodeRecords[T any](path string, destination *[]T) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return fmt.Errorf("read %q: no records", path)
	}
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, destination); err != nil {
			return fmt.Errorf("decode JSON array %q: %w", path, err)
		}
		return nil
	}
	lines := bytes.Split(trimmed, []byte{'\n'})
	for index, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var record T
		if err := json.Unmarshal(line, &record); err != nil {
			return fmt.Errorf("decode JSON Lines %q record %d: %w", path, index+1, err)
		}
		*destination = append(*destination, record)
	}
	return nil
}

func validateOccurrence(occurrence *domain.EventOccurrence) error {
	switch {
	case occurrence.ID == "":
		return errors.New("id is required")
	case occurrence.Type == "":
		return errors.New("type is required")
	case occurrence.MemberID == "":
		return errors.New("member_id is required")
	case occurrence.StartsAt.IsZero():
		return errors.New("starts_at is required")
	case occurrence.Duration < 0:
		return errors.New("duration must not be negative")
	default:
		return nil
	}
}

func recordError(path string, index int, err error) error {
	return fmt.Errorf("%s record %d: %w", path, index+1, err)
}

// EligibleForDuty reports whether a team member is eligible for a named duty.
func (member TeamMember) EligibleForDuty(duty string) bool {
	for _, eligibleDuty := range member.EligibleDuties {
		if strings.TrimSpace(eligibleDuty) == duty {
			return true
		}
	}
	return false
}
