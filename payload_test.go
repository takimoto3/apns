package apns_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/takimoto3/apns"
	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload"
)

// JSONComparer transforms JSON bytes into a map for semantic comparison.
var JSONComparer = cmp.Transformer("JSON", func(in []byte) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal(in, &m); err != nil {
		// Print JSON unmarshal error and return nil
		// This is treated as difference in comparison
		fmt.Printf("JSON Unmarshal Error: %v: json=%s\n", err, in)
		return nil
	}
	return m
})

func TestPayloadMarshalJSONTo3_FullCoverage(t *testing.T) {
	tests := map[string]struct {
		input apns.Payload
		want  string
	}{
		"empty": {
			input: apns.Payload{},
			want:  `{"aps":{}}`,
		},

		"empty APS struct": {
			input: apns.Payload{
				APS:        payload.APS{},
				CustomData: map[string]any{},
			},
			want: `{"aps":{}}`,
		},

		"APS with alert as string": {
			input: apns.Payload{
				APS: payload.APS{
					Alert: "simple alert",
				},
				CustomData: nil,
			},
			want: `{"aps":{"alert":"simple alert"}}`,
		},

		"APS with empty Alert struct": {
			input: apns.Payload{
				APS: payload.APS{
					Alert: payload.Alert{},
				},
			},
			want: `{"aps":{"alert":{}}}`,
		},

		"APS with Sound string": {
			input: apns.Payload{
				APS: payload.APS{
					Sound: "default",
				},
			},
			want: `{"aps":{"sound":"default"}}`,
		},

		"APS with empty Sound struct": {
			input: apns.Payload{
				APS: payload.APS{
					Sound: payload.Sound{},
				},
			},
			want: `{"aps":{"sound":{}}}`,
		},

		"APS with ContentAvailable and MutableContent": {
			input: apns.Payload{
				APS: payload.APS{
					ContentAvailable: 1,
					MutableContent:   1,
				},
			},
			want: `{"aps":{"content-available":1,"mutable-content":1}}`,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			gotBytes, err := json.Marshal(&tt.input)
			if err != nil {
				t.Fatalf("MarshalJSONTo3 error: %v", err)
			}

			// --- Unmarshal both to compare logical structure ---
			var got any
			if err := json.Unmarshal(gotBytes, &got); err != nil {
				t.Fatalf("custom JSON invalid: %v\nraw: %s", err, string(gotBytes))
			}

			var want any
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatalf("want JSON invalid: %v", err)
			}

			if diff := cmp.Diff(want, got, JSONComparer); diff != "" {
				t.Fatalf("custom JSON differs from want (-want +got):\n%s\ncustom: %s", diff, string(gotBytes))
			}
		})
	}
}

func TestPayloadMarshalJSONTo3_Realistic(t *testing.T) {
	tests := map[string]struct {
		input apns.Payload
		want  string
	}{
		"alert with sound and badge": {
			input: apns.Payload{
				APS: payload.APS{
					Alert: payload.Alert{
						Title:    "Hello",
						Subtitle: "Subtitle",
						Body:     "This is a notification",
					},
					Badge: 5,
					Sound: payload.Sound{
						Name:     "default",
						Critical: 1,
						Volume:   0.8,
					},
					Category: "MESSAGE_CATEGORY",
					ThreadID: "thread-123",
				},
				CustomData: map[string]any{
					"user_id": 42,
				},
			},
			want: `{
				"aps": {
					"alert": {
						"title":"Hello",
						"subtitle":"Subtitle",
						"body":"This is a notification"
					},
					"badge":5,
					"sound":{"name":"default","critical":1,"volume":0.8},
					"category":"MESSAGE_CATEGORY",
					"thread-id":"thread-123"
				},
				"user_id":42
			}`,
		},

		"mutable content with loc keys": {
			input: apns.Payload{
				APS: payload.APS{
					Alert: payload.Alert{
						LocKey:          "NOTIF_BODY",
						LocArgs:         []string{"Alice"},
						TitleLocKey:     "NOTIF_TITLE",
						TitleLocArgs:    []string{"Bob"},
						SubtitleLocKey:  "NOTIF_SUB",
						SubtitleLocArgs: []string{"Carol"},
						ActionLocKey:    "OPEN_APP",
					},
					MutableContent: 1,
				},
				CustomData: map[string]any{
					"extra": "value",
				},
			},
			want: `{
				"aps": {
					"alert": {
						"loc-key":"NOTIF_BODY",
						"loc-args":["Alice"],
						"title-loc-key":"NOTIF_TITLE",
						"title-loc-args":["Bob"],
						"subtitle-loc-key":"NOTIF_SUB",
						"subtitle-loc-args":["Carol"],
						"action-loc-key":"OPEN_APP"
					},
					"mutable-content":1
				},
				"extra":"value"
			}`,
		},

		"live activity update": {
			input: apns.Payload{
				APS: payload.APS{
					ContentState: map[string]any{
						"score": 10,
						"team":  "Eagles",
					},
					Event:           "update",
					TargetContentID: "activity-123",
				},
			},
			want: `{
				"aps": {
					"content-state": {
						"score":10,
						"team":"Eagles"
					},
					"event":"update",
					"target-content-id":"activity-123"
				}
			}`,
		},

		"live activity with timestamps": {
			input: apns.Payload{
				APS: payload.APS{
					Event:           "update",
					TargetContentID: "activity-456",
					StaleDate:       notification.NewEpochTime(time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)),
					Timestamp:       notification.NewEpochTime(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			want: `{
				"aps": {
					"event":"update",
					"target-content-id":"activity-456",
					"stale-date": 1672534800,
					"timestamp": 1672531200
				}
			}`,
		},

		"background push with content-available": {
			input: apns.Payload{
				APS: payload.APS{
					ContentAvailable: 1,
				},
				CustomData: map[string]any{
					"fetch_id": "abc-123",
				},
			},
			want: `{
				"aps": {
					"content-available":1
				},
				"fetch_id":"abc-123"
			}`,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			gotBytes, err := json.Marshal(&tt.input)
			if err != nil {
				t.Fatalf("MarshalJSONTo3 error: %v", err)
			}

			var got any
			if err := json.Unmarshal(gotBytes, &got); err != nil {
				t.Fatalf("custom JSON invalid: %v\nraw: %s", err, string(gotBytes))
			}

			// --- Compare with expected JSON ---
			var want any
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatalf("want JSON invalid: %v", err)
			}

			if diff := cmp.Diff(want, got, JSONComparer); diff != "" {
				t.Fatalf("custom JSON mismatch (-want +got):\n%s\ncustom: %s", diff, string(gotBytes))
			}
		})
	}
}
