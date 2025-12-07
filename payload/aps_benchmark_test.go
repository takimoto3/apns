package payload_test

import (
	"encoding/json"
	"testing"
	"time"

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

func BenchmarkAPSJSON_Full(b *testing.B) {
	aps := makeSampleAPS()

	b.Run("MarshalJSON(Standard)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(aps)
		}
	})
	b.Run("MarshalJSONFast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = aps.MarshalJSONFast()
		}
	})
}

func makeMinimalAPS() payload.APS {
	return payload.APS{
		Alert: payload.Alert{Title: "Hi"},
	}
}

func BenchmarkAPSJSON_Minimal(b *testing.B) {
	aps := makeMinimalAPS()
	b.Run("MarshalJSON(Standard)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(aps)
		}
	})
	b.Run("MarshalJSONFast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = aps.MarshalJSONFast()
		}
	})
}
