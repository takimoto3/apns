package apns_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/takimoto3/apns"
	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload"
	"github.com/takimoto3/apns/payload/interruptionlevel"
)

func makeSampleAPS() payload.APS {
	t := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	return payload.APS{
		Alert: payload.Alert{
			Title:           "Hello",
			Subtitle:        "Sub",
			Body:            "World",
			LaunchImage:     "img.png",
			LocKey:          "HELLO",
			LocArgs:         []string{"A", "B"},
			TitleLocKey:     "TITLE",
			TitleLocArgs:    []string{"X", "Y"},
			SubtitleLocKey:  "SUB",
			SubtitleLocArgs: []string{"C"},
			ActionLocKey:    "ACTION",
		},
		Badge:             5,
		Sound:             payload.Sound{Name: "ping.aiff", Critical: 1, Volume: 0.8},
		ContentAvailable:  1,
		MutableContent:    1,
		Category:          "news",
		ThreadID:          "thread123",
		InterruptionLevel: interruptionlevel.Active,
		RelevanceScore:    0.9,
		StaleDate:         notification.NewEpochTime(t.Add(60 * time.Second)),
		Timestamp:         notification.NewEpochTime(t),
		FilterCriteria:    "important",
		TargetContentID:   "activity123",
		ContentState:      map[string]any{"state": "running"},
		Event:             "start",
		DismissalDate:     1699999999,
		AttributesType:    "LiveActivity",
		Attributes:        map[string]any{"key": "value"},
	}
}

func makeSamplePayload() apns.Payload {
	return apns.Payload{
		APS: makeSampleAPS(),
		CustomData: map[string]any{
			"action": "open",
			"id":     123,
			"active": true,
			"score":  0.95,
			"nested": map[string]any{
				"foo": "bar",
			},
		},
	}
}

func makeMinimalPayload() apns.Payload {
	return apns.Payload{
		APS: payload.APS{
			Alert: payload.Alert{
				Body: "Hello",
			},
		},
		CustomData: nil,
	}
}

func BenchmarkPayloadJSON(b *testing.B) {
	full := makeSamplePayload()
	minimal := makeMinimalPayload()

	// --- Full ---
	b.Run("Full/MarshalJSON(Standard)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(full)
		}
	})
	b.Run("Full/MarshalJSONFast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = full.MarshalJSONFast()
		}
	})

	// --- Minimal ---
	b.Run("Minimal/MarshalJSON(Standard)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(minimal)
		}
	})
	b.Run("Minimal/MarshalJSONFast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = minimal.MarshalJSONFast()
		}
	})
}

func BenchmarkPushTypePayloads(b *testing.B) {
	for pushType, pl := range pushTypePayloads {
		b.Run(pushType+"/MarshalJSON(Standard)", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = json.Marshal(pl)
			}
		})

		b.Run(pushType+"/MarshalJSONFast", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = pl.MarshalJSONFast()
			}
		})
	}
}

// ------------------------------
// Sample Payloads per Push Type
// ------------------------------
// These payloads are used for testing, benchmarking, and documentation purposes.
// Each entry represents a realistic example for a specific APNs push type.
var pushTypePayloads = map[string]apns.Payload{
	"alert": {
		APS: payload.APS{
			Alert: payload.Alert{
				Title:   "New Message",
				Body:    "You have a new message",
				LocKey:  "NEW_MSG",
				LocArgs: []string{"Alice"},
			},
			Badge:            3,
			Sound:            &payload.Sound{Name: "default", Critical: 1, Volume: 1.0},
			ContentAvailable: 0,
			MutableContent:   1,
			Category:         "MESSAGE",
		},
		CustomData: map[string]any{
			"conversation_id": 12345,
		},
	},

	"background": {
		APS: payload.APS{
			ContentAvailable: 1,
		},
		CustomData: map[string]any{
			"sync_token": "abc123",
		},
	},

	"voip": {
		APS: payload.APS{
			Alert: payload.Alert{
				Title: "Incoming Call",
				Body:  "Bob is calling",
			},
			MutableContent: 1,
		},
		CustomData: map[string]any{
			"call_id":   "call-001",
			"caller":    "Bob",
			"timestamp": 1699999999,
		},
	},

	"pushtotalk": {
		APS: payload.APS{
			MutableContent: 1,
		},
		CustomData: map[string]any{
			"session_id": "ptt-001",
			"user":       "Alice",
			"action":     "speak",
		},
	},

	"location": {
		APS: payload.APS{
			ContentAvailable: 1,
		},
		CustomData: map[string]any{
			"region_id": "geo-123",
			"radius":    50,
		},
	},

	"widgets": {
		APS: payload.APS{
			MutableContent: 1,
		},
		CustomData: map[string]any{
			"widget_id": "weather_widget",
			"update_at": 1700000000,
		},
	},

	"complication": {
		APS: payload.APS{
			MutableContent: 1,
		},
		CustomData: map[string]any{
			"complication_id": "step_count",
			"value":           1234,
		},
	},

	"controls": {
		APS: payload.APS{
			MutableContent: 1,
		},
		CustomData: map[string]any{
			"control_id": "lamp_01",
			"state":      "on",
		},
	},

	"fileprovider": {
		APS: payload.APS{
			ContentAvailable: 1,
		},
		CustomData: map[string]any{
			"file_id":   "file_001",
			"action":    "update",
			"timestamp": 1700001234,
		},
	},

	"liveactivity": {
		APS: payload.APS{
			Alert: payload.Alert{
				Title: "Match Score Update",
			},
			ContentState: map[string]any{
				"home_score": 2,
				"away_score": 1,
			},
			Event:          "update",
			AttributesType: "MatchActivityAttributes",
			Attributes: map[string]any{
				"match_id":  "match-001",
				"home_team": "Team A",
				"away_team": "Team B",
			},
		},
	},

	"mdm": {
		APS: payload.APS{}, // APS is optional for MDM push payloads
		CustomData: map[string]any{
			"mdm": "base64-encoded-command",
		},
	},
}
