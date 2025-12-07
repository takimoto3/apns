package payload_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestSoundMarshalJSONTo3(t *testing.T) {
	tests := map[string]struct {
		input payload.Sound
		want  string
	}{
		"all fields": {
			input: payload.Sound{
				Critical: 1,
				Name:     "alert",
				Volume:   0.8,
			},
			want: `{"critical":1,"name":"alert","volume":0.8}`,
		},
		"only name": {
			input: payload.Sound{
				Name: "ping",
			},
			want: `{"name":"ping"}`,
		},
		"only critical": {
			input: payload.Sound{
				Critical: 2,
			},
			want: `{"critical":2}`,
		},
		"empty struct": {
			input: payload.Sound{},
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

			// --- Standard JSON using alias (no MarshalJSONTo3 is called) ---
			// Alias type to avoid calling MarshalJSONTo3.
			type soundAlias payload.Sound
			gotStd, err := json.Marshal(soundAlias(tt.input))
			if err != nil {
				t.Fatalf("standard json.Marshal error: %v", err)
			}

			// --- Compare against expected JSON ---
			if diff := cmp.Diff([]byte(tt.want), gotCustom, JSONComparer); diff != "" {
				t.Errorf("custom JSON mismatch (-want +got):\n%s", diff)
			}

			// --- Compare custom JSON vs standard JSON ---
			if diff := cmp.Diff(gotStd, gotCustom, JSONComparer); diff != "" {
				t.Errorf("custom JSON differs from standard JSON (-std +custom):\n%s", diff)
			}
		})
	}
}
