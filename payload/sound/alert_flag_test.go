package sound_test

import (
	"testing"

	"github.com/takimoto3/apns/payload/sound"
)

func TestAlertFlag_Values(t *testing.T) {
	testCases := []struct {
		name     string
		flag     sound.AlertFlag
		expected int
	}{
		{
			name:     "None",
			flag:     sound.None,
			expected: 0,
		},
		{
			name:     "Critical",
			flag:     sound.Critical,
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if int(tc.flag) != tc.expected { // Directly compare the underlying int value
				t.Errorf("AlertFlag %s got %d, want %d", tc.name, int(tc.flag), tc.expected)
			}
		})
	}
}
