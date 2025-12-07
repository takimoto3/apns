package notification_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/takimoto3/apns/notification"
)

func TestNewEpochTime(t *testing.T) {
	testCases := map[string]struct {
		inputTime time.Time
		expected  int64
	}{
		"Zero time": {
			inputTime: time.Time{},
			expected:  0,
		},
		"Specific time": {
			inputTime: time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC),
			expected:  1,
		},
		"Another specific time": {
			inputTime: time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC),
			expected:  1698400800,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			epochTime := notification.NewEpochTime(tc.inputTime)
			if int64(*epochTime) != tc.expected {
				t.Errorf("NewEpochTime(%v) = %d; want %d", tc.inputTime, *epochTime, tc.expected)
			}
		})
	}
}

func TestEpochTimeString(t *testing.T) {
	testCases := map[string]struct {
		epochTime notification.EpochTime
		expected  string
	}{
		"Zero": {
			epochTime: 0,
			expected:  "0",
		},
		"Positive number": {
			epochTime: 12345,
			expected:  "12345",
		},
		"Large number": {
			epochTime: 1698397200,
			expected:  "1698397200",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			str := tc.epochTime.String()
			if str != tc.expected {
				t.Errorf("EpochTime(%d).String() = %s; want %s", tc.epochTime, str, tc.expected)
			}
			// Also test if it can be parsed back
			parsed, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				t.Fatalf("Failed to parse string back to int64: %v", err)
			}
			if parsed != int64(tc.epochTime) {
				t.Errorf("Parsed value %d does not match original epoch time %d", parsed, tc.epochTime)
			}
		})
	}
}
