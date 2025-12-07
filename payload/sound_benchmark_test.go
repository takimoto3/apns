package payload_test

import (
	"encoding/json"
	"testing"

	"github.com/takimoto3/apns/payload"
	"github.com/takimoto3/apns/payload/sound"
)

func BenchmarkSoundMarshal(b *testing.B) {
	s := &payload.Sound{
		Name:     "bingbong.aiff",
		Critical: sound.Critical,
		Volume:   0.8,
	}

	b.Run("json.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(s)
		}
	})

	b.Run("MarshalJSONFast", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = s.MarshalJSONFast()
		}
	})
}
