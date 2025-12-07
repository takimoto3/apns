package interruptionlevel_test

import (
	"testing"

	"github.com/takimoto3/apns/payload/interruptionlevel"
)

func TestInterruptionLevel_Values(t *testing.T) {
	testCases := []struct {
		name     string
		level    interruptionlevel.InterruptionLevel
		expected string
	}{
		{
			name:     "Passive",
			level:    interruptionlevel.Passive,
			expected: "passive",
		},
		{
			name:     "Active",
			level:    interruptionlevel.Active,
			expected: "active",
		},
		{
			name:     "TimeSensitive",
			level:    interruptionlevel.TimeSensitive,
			expected: "time-sensitive",
		},
		{
			name:     "Critical",
			level:    interruptionlevel.Critical,
			expected: "critical",
		},
		// You might also want to test an unknown string to ensure it behaves as a simple string,
		// though the InterruptionLevel type itself doesn't have validation logic.
		{
			name:     "Unknown",
			level:    interruptionlevel.InterruptionLevel("unknown-level"),
			expected: "unknown-level", // It should just return its own string value
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.level) != tc.expected { // Directly compare the underlying string value
				t.Errorf("InterruptionLevel %s got %q, want %q", tc.name, string(tc.level), tc.expected)
			}
		})
	}
}
