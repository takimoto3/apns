package payload_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takimoto3/apns/payload"
)

func TestAlertMarshalJSONTo3(t *testing.T) {

	tests := map[string]struct {
		input payload.Alert
		want  string
	}{
		"all fields": {
			input: payload.Alert{
				Title:           "Game Request",
				Subtitle:        "Five Card Draw",
				Body:            "Bob wants to play",
				LaunchImage:     "img.png",
				LocKey:          "GAME_PLAY_REQUEST_FORMAT",
				LocArgs:         []string{"Bob"},
				TitleLocKey:     "GAME_TITLE_KEY",
				TitleLocArgs:    []string{"Bob"},
				SubtitleLocKey:  "GAME_SUB_KEY",
				SubtitleLocArgs: []string{"Bob"},
				ActionLocKey:    "PLAY",
			},
			want: `{
				"title":"Game Request",
				"subtitle":"Five Card Draw",
				"body":"Bob wants to play",
				"launch-image":"img.png",
				"loc-key":"GAME_PLAY_REQUEST_FORMAT",
				"loc-args":["Bob"],
				"title-loc-key":"GAME_TITLE_KEY",
				"title-loc-args":["Bob"],
				"subtitle-loc-key":"GAME_SUB_KEY",
				"subtitle-loc-args":["Bob"],
				"action-loc-key":"PLAY"
			}`,
		},

		"only title": {
			input: payload.Alert{
				Title: "Hello",
			},
			want: `{"title":"Hello"}`,
		},

		"with empty slices": {
			input: payload.Alert{
				Title:        "Test",
				LocArgs:      []string{},
				TitleLocArgs: []string{},
			},
			want: `{"title":"Test"}`,
		},

		"escaping check": {
			input: payload.Alert{
				Body: `He said "Hi"`,
			},
			want: `{"body":"He said \"Hi\""}`,
		},

		"empty struct": {
			input: payload.Alert{},
			want:  `{}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			// --- Custom JSON ---
			gotCustom, err := tt.input.MarshalJSONFast()
			if err != nil {
				t.Fatalf("MarshalJSONTo3 error: %v", err)
			}

			checker := newDuplicateKeyChecker(gotCustom)
			if err := checker.check(); err != nil {
				t.Errorf("Duplicate key check failed: %v\nJSON: %s", err, string(gotCustom))
				return
			}

			// --- Standard JSON (via alias = no MarshalJSONTo3) ---
			// Alias to avoid invoking MarshalJSONTo3
			type alertAlias payload.Alert
			gotStd, err := json.Marshal(alertAlias(tt.input))
			if err != nil {
				t.Fatalf("standard json.Marshal error: %v", err)
			}

			// --- Compare custom JSON to expected JSON ---
			if diff := cmp.Diff([]byte(tt.want), gotCustom, JSONComparer); diff != "" {
				t.Errorf("custom JSON mismatch (-want +got):\n%s", diff)
			}

			// --- Compare custom JSON to standard JSON ---
			if diff := cmp.Diff(gotStd, gotCustom, JSONComparer); diff != "" {
				t.Errorf("custom JSON differs from standard JSON (-std +custom):\n%s", diff)
			}
		})
	}
}
