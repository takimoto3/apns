package payload_test

import (
	"strings"
	"testing"

	"github.com/takimoto3/apns/payload"
)

func TestSoundValidate(t *testing.T) {
	tests := map[string]struct {
		sound         payload.Sound
		wantErrString string // If non-empty, an error is expected, and this string should be in the error message
	}{
		"valid_minimal_sound": {
			sound:         payload.Sound{Name: "default"},
			wantErrString: "",
		},
		"valid_full_sound": {
			sound:         payload.Sound{Name: "alarm.aiff", Critical: 1, Volume: 0.5},
			wantErrString: "",
		},
		"valid_sound_critical_zero": {
			sound:         payload.Sound{Name: "default", Critical: 0, Volume: 0.8},
			wantErrString: "",
		},
		"invalid_critical_value": {
			sound:         payload.Sound{Name: "default", Critical: 2}, // Critical can only be 0 or 1
			wantErrString: "invalid critical flag: 2",
		},
		"invalid_volume_too_low": {
			sound:         payload.Sound{Name: "default", Volume: -0.1}, // Volume must be 0.0-1.0
			wantErrString: "volume field error: ratio out of range: -0.100000",
		},
		"invalid_volume_too_high": {
			sound:         payload.Sound{Name: "default", Volume: 1.1}, // Volume must be 0.0-1.0
			wantErrString: "volume field error: ratio out of range: 1.100000",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.sound.Validate()
			if err != nil {
				if tt.wantErrString == "" {
					t.Errorf("Sound.Validate() returned unexpected error: %v", err)
				} else if !strings.Contains(err.Error(), tt.wantErrString) {
					t.Errorf("Sound.Validate() error = %v, wantErrString '%s'", err, tt.wantErrString)
				}
			} else {
				if tt.wantErrString != "" {
					t.Errorf("Sound.Validate() expected an error containing %q, but got none", tt.wantErrString)

				}
			}
		})
	}
}
