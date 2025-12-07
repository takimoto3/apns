package priority_test

import (
	"testing"

	"github.com/takimoto3/apns/notification/priority"
)

func TestPriority_String(t *testing.T) {
	testCases := map[string]struct {
		priority priority.Priority
		expected string
	}{
		"PowerOnly": {
			priority: priority.PowerOnly,
			expected: "1",
		},
		"Conserve": {
			priority: priority.Conserve,
			expected: "5",
		},
		"Immediate": {
			priority: priority.Immediate,
			expected: "10",
		},
		"None": {
			priority: priority.None,
			expected: "",
		},
		"Undefined": {
			priority: 99,
			expected: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			str := tc.priority.String()
			if str != tc.expected {
				t.Errorf("Priority(%d).String() = %q; want %q", tc.priority, str, tc.expected)
			}
		})
	}
}
