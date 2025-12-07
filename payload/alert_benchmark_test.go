package payload_test

import (
	"encoding/json"
	"testing"

	"github.com/takimoto3/apns/payload"
)

func BenchmarkAlertJSON(b *testing.B) {
	alert := payload.Alert{
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
	}

	b.Run("encoding_json", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(alert)
		}
	})

	b.Run("MarshalJSONTo3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = alert.MarshalJSONFast()
		}
	})

}
