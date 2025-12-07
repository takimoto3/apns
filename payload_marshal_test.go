package apns_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/takimoto3/apns"
	"github.com/takimoto3/apns/payload"
)

// PayloadAlias is used to force json.Marshal to use the default encoder
// (because the alias type does not have MarshalJSONTo3).
type PayloadAlias apns.Payload

// MarshalJSON merges APS and CustomData at the root level for JSON output.
func (p PayloadAlias) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	m["aps"] = p.APS
	for k, v := range p.CustomData {
		m[k] = v
	}
	return json.Marshal(m)
}

func TestPayloadMarshalJSONTo3(t *testing.T) {
	tests := map[string]struct {
		input apns.Payload
		want  string
	}{
		"only aps": {
			input: apns.Payload{
				APS: payload.APS{
					Alert: payload.Alert{
						Title: "Hello",
					},
				},
			},
			want: `{
				"aps": {
					"alert": {
						"title":"Hello"
					}
				}
			}`,
		},

		"aps + simple custom": {
			input: apns.Payload{
				APS: payload.APS{
					ContentAvailable: 1,
				},
				CustomData: map[string]any{
					"a": "text",
					"b": 123,
					"c": true,
				},
			},
			want: `{
				"aps": {"content-available":1},
				"a": "text",
				"b": 123,
				"c": true
			}`,
		},

		"aps + nested object": {
			input: apns.Payload{
				APS: payload.APS{
					MutableContent: 1,
				},
				CustomData: map[string]any{
					"meta": map[string]any{
						"id":   10,
						"name": "foo",
					},
				},
			},
			want: `{
				"aps": {"mutable-content":1},
				"meta": {
					"id":10,
					"name":"foo"
				}
			}`,
		},

		"aps + array values": {
			input: apns.Payload{
				APS: payload.APS{
					ContentAvailable: 1,
				},
				CustomData: map[string]any{
					"arr": []any{"a", 1, true, nil},
				},
			},
			want: `{
				"aps":{"content-available":1},
				"arr":["a",1,true,null]
			}`,
		},

		"empty custom data": {
			input: apns.Payload{
				APS: payload.APS{
					Alert: payload.Alert{Body: "body"},
				},
				CustomData: map[string]any{},
			},
			want: `{
				"aps":{"alert":{"body":"body"}}
			}`,
		},

		"nil custom data": {
			input: apns.Payload{
				APS:        payload.APS{},
				CustomData: nil,
			},
			want: `{"aps":{}}`,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {

			// --- Custom JSON ---
			gotBytes, err := tt.input.MarshalJSONFast()
			if err != nil {
				t.Fatalf("MarshalJSONTo3 error: %v", err)
			}

			// Semantic comparison with want using JSONComparer
			if diff := cmp.Diff([]byte(tt.want), gotBytes, JSONComparer); diff != "" {
				t.Fatalf("custom JSON mismatch (-want +got):\n%s\nraw: %s", diff, string(gotBytes))
			}

			// --- Standard JSON using alias (no MarshalJSON) ---
			gotStd, err := json.Marshal(PayloadAlias(tt.input))
			if err != nil {
				t.Fatalf("standard json.Marshal error: %v", err)
			}

			// Compare custom JSON with standard JSON semantically
			if diff := cmp.Diff(gotStd, gotBytes, JSONComparer); diff != "" {
				t.Fatalf("custom JSON differs from standard JSON (-std +custom):\n%s\ncustom: %s\nstd: %s",
					diff, string(gotBytes), string(gotStd))
			}
		})
	}
}
