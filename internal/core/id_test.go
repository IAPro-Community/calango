package core

import (
	"strings"
	"testing"
	"time"
)

func TestNewIDIsULIDShapedAndSortsByTime(t *testing.T) {
	earlier, err := NewID(time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewID() error = %v", err)
	}
	later, err := NewID(time.Date(2026, 11, 23, 12, 0, 0, 1_000_000, time.UTC))
	if err != nil {
		t.Fatalf("NewID() error = %v", err)
	}

	for _, id := range []string{earlier, later} {
		if len(id) != 26 {
			t.Fatalf("NewID() = %q, want 26 characters", id)
		}
		for _, r := range id {
			if !strings.ContainsRune(crockfordAlphabet, r) {
				t.Fatalf("NewID() = %q contains %q outside Crockford base32", id, r)
			}
		}
	}
	if earlier >= later {
		t.Fatalf("NewID(earlier) = %q should sort before NewID(later) = %q", earlier, later)
	}
}
