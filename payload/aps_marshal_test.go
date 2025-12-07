package payload_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload"
	"github.com/takimoto3/apns/payload/interruptionlevel"
)

func TestAPSMarshalJSONTo3(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(3600 * time.Second)

	tests := map[string]struct {
		input payload.APS
		want  string
	}{
		"empty APS": {
			input: payload.APS{},
			want:  `{}`,
		},
		"simple alert string": {
			input: payload.APS{
				Alert: "Hello",
			},
			want: `{"alert":"Hello"}`,
		},
		"alert object(not pointer)": {
			input: payload.APS{
				Alert: payload.Alert{
					Title: "Hi",
				},
			},
			want: `{"alert":{"title":"Hi"}}`,
		},
		"alert object + badge + sound string": {
			input: payload.APS{
				Alert: &payload.Alert{
					Title: "Hi",
				},
				Badge: 5,
				Sound: "default",
			},
			want: `{
				"alert":{"title":"Hi"},
				"badge":5,
				"sound":"default"
			}`,
		},
		"sound object(not pointer)": {
			input: payload.APS{
				Sound: payload.Sound{
					Name: "beep",
				},
			},
			want: `{"sound":{"name":"beep"}}`,
		},
		"sound object + category + thread-id": {
			input: payload.APS{
				Sound: &payload.Sound{
					Name: "beep",
				},
				Category: "GAME",
				ThreadID: "thread-1",
			},
			want: `{
				"sound":{"name":"beep"},
				"category":"GAME",
				"thread-id":"thread-1"
			}`,
		},

		"content-available + mutable-content": {
			input: payload.APS{
				ContentAvailable: 1,
				MutableContent:   1,
			},
			want: `{
				"content-available":1,
				"mutable-content":1
			}`,
		},

		"filter-criteria & content-state": {
			input: payload.APS{
				FilterCriteria: "important",
				ContentState: map[string]any{
					"score":  150,
					"status": "ok",
					"ratio":  0.75,
				},
			},
			want: `{
				"filter-criteria":"important",
				"content-state":{
					"score":150,
					"status":"ok",
					"ratio":0.75
				}
			}`,
		},

		"event + dismissal-date + attributes": {
			input: payload.APS{
				Event:          "update",
				DismissalDate:  1737200000,
				AttributesType: "meta",
				Attributes: map[string]any{
					"id":    "ABC",
					"count": 3,
				},
			},
			want: `{
				"event":"update",
				"dismissal-date":1737200000,
				"attributes-type":"meta",
				"attributes":{
					"id":"ABC",
					"count":3
				}
			}`,
		},
		"APS full data": {
			input: payload.APS{
				Alert: payload.Alert{
					Title: "Game Request",
					Body:  "Bob wants to play poker",
				},
				Badge:             5,
				Sound:             "default",
				ContentAvailable:  1,
				MutableContent:    1,
				Category:          "GAME_CATEGORY",
				ThreadID:          "thread-123",
				InterruptionLevel: interruptionlevel.Passive,
				RelevanceScore:    0.8,
				StaleDate:         notification.NewEpochTime(t2),
				Timestamp:         notification.NewEpochTime(t1),
				FilterCriteria:    "important",
				TargetContentID:   "activity-1",
				ContentState: map[string]any{
					"score": 100,
				},
				Event:          "update",
				DismissalDate:  1700000000,
				AttributesType: "GameAttributes",
				Attributes: map[string]any{
					"level": 10,
				},
			},
			want: `{
				"alert":{
					"title":"Game Request",
					"body":"Bob wants to play poker"
				},
				"badge":5,
				"sound":"default",
				"content-available":1,
				"mutable-content":1,
				"category":"GAME_CATEGORY",
				"thread-id":"thread-123",
				"interruption-level":"passive",
				"relevance-score":0.8,
				"stale-date":1672534800,
				"timestamp":1672531200,
				"filter-criteria":"important",
				"target-content-id":"activity-1",
				"content-state":{"score":100},
				"event":"update",
				"dismissal-date":1700000000,
				"attributes-type":"GameAttributes",
				"attributes":{"level":10}
			}`,
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

			// --- Standard JSON using alias (no MarshalJSONTo3) ---
			// Alias to avoid calling MarshalJSONTo3.
			type apsAlias payload.APS
			gotStd, err := json.Marshal(apsAlias(tt.input))
			if err != nil {
				t.Fatalf("standard json.Marshal error: %v", err)
			}

			// --- Compare with expected JSON ---
			if diff := cmp.Diff([]byte(tt.want), gotCustom, JSONComparer); diff != "" {
				t.Errorf("custom JSON mismatch (-want +got):\n%s", diff)
			}

			// --- Compare custom JSON with standard JSON ---
			if diff := cmp.Diff(gotStd, gotCustom, JSONComparer); diff != "" {
				t.Errorf("custom JSON differs from standard JSON (-std +custom):\n%s", diff)
			}
		})
	}
}

// --- Duplicate Key Checker Logic ---

type duplicateKeyChecker struct {
	dec *json.Decoder
}

func newDuplicateKeyChecker(data []byte) *duplicateKeyChecker {
	return &duplicateKeyChecker{dec: json.NewDecoder(bytes.NewReader(data))}
}

func (c *duplicateKeyChecker) check() error {
	// The top-level JSON must be an object for APS
	t, err := c.dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected top-level object, got %T %v", t, t)
	}
	return c.checkObject() // Start checking from the root object
}

func (c *duplicateKeyChecker) checkObject() error {
	keys := make(map[string]bool)
	for c.dec.More() {
		// Read the key
		t, err := c.dec.Token()
		if err != nil {
			return err
		}
		key, ok := t.(string)
		if !ok {
			return fmt.Errorf("expected string key, got %T %v", t, t)
		}
		if keys[key] {
			return fmt.Errorf("duplicate key found: %s", key)
		}
		keys[key] = true

		// Consume the value associated with the key
		if err := c.consumeValue(); err != nil {
			return err
		}
	}
	// Consume closing '}'
	t, err := c.dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '}' {
		return fmt.Errorf("expected end of object '}', got %T %v", t, t)
	}
	return nil
}

// consumeValue consumes the next value from the decoder, recursing into objects and arrays.
func (c *duplicateKeyChecker) consumeValue() error {
	t, err := c.dec.Token()
	if err != nil {
		return err
	}

	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			return c.checkObject() // Recursively check nested object
		case '[':
			for c.dec.More() {
				if err := c.consumeValue(); err != nil {
					return err
				}
			}
			// Consume the closing ']'
			if _, err := c.dec.Token(); err != nil {
				return fmt.Errorf("failed to read end of array token: %w", err)
			}
		}
	}
	return nil
}

// MockMarshaler is a simple type that implements json.Marshaler
type MockMarshaler struct {
	Value string
}

func (m MockMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s_marshaled"`, m.Value)), nil
}

func TestEncodeValue(t *testing.T) {
	tms := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    any
		expected string
		wantErr  bool
	}{
		{name: "nil", input: nil, expected: "null", wantErr: false},
		{name: "string", input: "hello", expected: `"hello"`, wantErr: false},
		{name: "int_positive", input: 123, expected: "123", wantErr: false},
		{name: "int_negative", input: -45, expected: "-45", wantErr: false},
		{name: "int_zero", input: 0, expected: "0", wantErr: false},
		{name: "int64", input: int64(10), expected: "10", wantErr: false},
		{name: "float64_positive", input: 123.45, expected: "123.45", wantErr: false},
		{name: "float64_negative", input: -67.89, expected: "-67.89", wantErr: false},
		{name: "float64_zero", input: 0.0, expected: "0", wantErr: false}, // Note: Go's json.Marshal for 0.0 outputs "0"
		{name: "bool_true", input: true, expected: "true", wantErr: false},
		{name: "bool_false", input: false, expected: "false", wantErr: false},
		{name: "byte_slice", input: []byte("test"), expected: `"test"`, wantErr: false},
		{name: "empty_byte_slice", input: []byte{}, expected: `""`, wantErr: false},
		{name: "slice_of_strings", input: []string{"a", "b"}, expected: `["a","b"]`, wantErr: false},
		{name: "slice_of_ints", input: []int{1, 2}, expected: `[1,2]`, wantErr: false},
		{name: "slice_of_int64", input: []int64{1, 2}, expected: `[1,2]`, wantErr: false},
		{name: "slice_of_float64", input: []float64{1.0, 1.5, 2}, expected: `[1.0, 1.5, 2.0]`, wantErr: false},
		{name: "slice_of_any", input: []any{"a", 1, true}, expected: `["a",1,true]`, wantErr: false},
		{name: "empty_slice", input: []any{}, expected: `[]`, wantErr: false},
		{name: "map_string_any", input: map[string]any{"key": "value", "num": 123}, expected: `{"key":"value","num":123}`, wantErr: false},
		{name: "empty_map", input: map[string]any{}, expected: `{}`, wantErr: false},
		{name: "json_marshaler_impl", input: MockMarshaler{Value: "custom"}, expected: `"custom_marshaled"`, wantErr: false},
		{name: "epoch_time", input: notification.EpochTime(tms.Unix()), expected: fmt.Sprintf(`%d`, tms.Unix()), wantErr: false},
		{name: "pointer_to_epoch_time", input: notification.NewEpochTime(tms), expected: fmt.Sprintf(`%d`, tms.Unix()), wantErr: false},
		// Test cases that might cause errors in custom encoder or are not supported
		{name: "unsupported_type_func", input: func() {}, expected: "", wantErr: true},
		{name: "unsupported_type_chan", input: make(chan int), expected: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := payload.EncodeValue(nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr {
				return
			}

			var got any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("invalid JSON from EncodeValue: %v", err)
			}

			var want any
			if err := json.Unmarshal([]byte(tt.expected), &want); err != nil {
				t.Fatalf("invalid expected JSON: %v", err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
