package payload_test

import (
	"strings"
	"testing"
	"time"

	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload"
	"github.com/takimoto3/apns/payload/interruptionlevel"
)

func TestAPSValidate(t *testing.T) {
	// Fixed times for deterministic tests
	tms1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	tms2 := tms1.Add(3600 * time.Second) // t1 + 1 hour

	tests := map[string]struct {
		aps           payload.APS
		wantErrString string // If non-empty, an error is expected, and this string should be in the error message
	}{

		"valid_minimal_alert": {
			aps: payload.APS{
				Alert: "Hello",
			},
			wantErrString: "",
		},

		"valid_relevance_score_live_activity_high": {
			aps: payload.APS{
				Event:          "update",
				ContentState:   map[string]any{"status": "running"},
				RelevanceScore: 25.0, // Valid for Live Activity (can be > 1.0)
			},
			wantErrString: "",
		},
		"valid_full_aps": {
			aps: payload.APS{
				Alert:             &payload.Alert{Title: "Title", Body: "Body"},
				Badge:             1,
				Sound:             &payload.Sound{Name: "default", Critical: 1, Volume: 1.0},
				ContentAvailable:  1,
				MutableContent:    1,
				Category:          "category",
				ThreadID:          "thread-id",
				InterruptionLevel: interruptionlevel.Active,
				RelevanceScore:    0.5,
				StaleDate:         notification.NewEpochTime(tms2),
				FilterCriteria:    "focus",
				Timestamp:         notification.NewEpochTime(tms1),
				TargetContentID:   "content-id",
				ContentState:      map[string]any{"key": "value"},
				Event:             "update",
				DismissalDate:     tms1.Unix(),
				AttributesType:    "type",
				Attributes:        map[string]any{"attr": 1},
			},
			wantErrString: "",
		},
		"invalid_empty_aps": {
			aps:           payload.APS{},
			wantErrString: "aps dictionary must not be empty",
		},

		"invalid_alert_type": {
			aps: payload.APS{
				Alert: 123, // Should be string, Alert, or *Alert
			},
			wantErrString: "invalid type for aps.Alert",
		},
		"invalid_badge_type_string": {
			aps: payload.APS{
				Badge: "invalid", // Should be int
			},
			wantErrString: "invalid type for aps.Badge",
		},
		"invalid_badge_type_float_non_int": {
			aps: payload.APS{
				Badge: 1.5, // Should be int
			},
			wantErrString: "invalid type for aps.Badge",
		},
		"invalid_sound_type": {
			aps: payload.APS{
				Sound: true, // Should be string, Sound, or *Sound
			},
			wantErrString: "invalid type for aps.Sound",
		},
		"invalid_content_available_value": {
			aps: payload.APS{
				ContentAvailable: 0, // Should be 1
			},
			wantErrString: "invalid value for aps.ContentAvailable",
		},
		"invalid_mutable_content_value": {
			aps: payload.APS{
				MutableContent: 2, // Should be 1
			},
			wantErrString: "invalid value for aps.MutableContent",
		},
		"invalid_interruption_level_string": {
			aps: payload.APS{
				Alert:             "Hello",
				InterruptionLevel: "unknown", // Not a valid enum
			},
			wantErrString: "invalid value for aps.InterruptionLevel",
		},
		"invalid_event_string": {
			aps: payload.APS{
				ContentState: map[string]any{"status": "running"},
				Event:        "bad-event", // Not "start", "update", or "end"
			},
			wantErrString: "invalid value for aps.Event",
		},
		"invalid_relevance_score_type": {
			aps: payload.APS{
				Alert:          "Hello",
				RelevanceScore: "not a number", // Should be a number
			},
			wantErrString: "invalid type for aps.RelevanceScore",
		},
		"relevance_score_standard": {
			aps: payload.APS{
				Alert:          "Hello",
				RelevanceScore: 1.0,
			},
			wantErrString: "",
		},
		"relevance_score_out_of_range_standard": {
			aps: payload.APS{
				Alert:          "Hello",
				RelevanceScore: 1.1, // > 1.0 for standard (not Live Activity)
			},
			wantErrString: "relevance-score must be between 0.0 and 1.0",
		},
		"relevance_score_negative_standard": {
			aps: payload.APS{
				Alert:          "Hello",
				RelevanceScore: -0.1, // < 0.0 for standard (not Live Activity)
			},
			wantErrString: "relevance-score must be between 0.0 and 1.0",
		},
		"relevance_score_liveactivity": {
			aps: payload.APS{
				ContentState:   map[string]any{"status": "running"},
				RelevanceScore: 1.0,
			},
			wantErrString: "",
		},
		"relevance_score_out_of_range_liveactivity": {
			aps: payload.APS{
				ContentState:   map[string]any{"status": "running"},
				RelevanceScore: 1.1, // > 1.0 for standard (not Live Activity)
			},
			wantErrString: "",
		},
		"relevance_score_negative_liveactivity": {
			aps: payload.APS{
				ContentState:   map[string]any{"status": "running"},
				RelevanceScore: -0.1, // < 0.0 for standard (not Live Activity)
			},
			wantErrString: "",
		},

		"sound_validate_error": { // Test that nested Validate() is called
			aps: payload.APS{
				Sound: payload.Sound{Name: "default", Critical: 2}, // Critical != 1
			},
			wantErrString: "invalid critical flag: 2", // Assuming Sound.Validate() returns this
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.aps.Validate()

			if err != nil { // An error occurred
				if tt.wantErrString == "" { // But no error was expected
					t.Errorf("APS.Validate() returned unexpected error: %v", err)
				} else if !strings.Contains(err.Error(), tt.wantErrString) { // Error was expected, but message content is wrong
					t.Errorf("APS.Validate() error message = %q, want it to contain %q", err.Error(), tt.wantErrString)
				}
			} else { // No error occurred
				if tt.wantErrString != "" { // But an error was expected
					t.Errorf("APS.Validate() expected an error containing %q, but got none", tt.wantErrString)
				}
			}
		})
	}
}
