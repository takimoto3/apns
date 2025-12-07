//go:build !use_std_json
// +build !use_std_json

// package payload provides types for constructing the payload of an APNs notification.
package payload

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	"github.com/takimoto3/apns/notification"
)

// ErrInvalidType is returned when a field in the APS dictionary has a type that
// cannot be marshaled by the custom JSON encoder.
var ErrInvalidType = errors.New("invalid type for APS field")

var apsBufSize = 560

var apsPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, apsBufSize)
		return &b
	},
}

// MarshalJSONFast is a custom JSON marshaler for the APS type that is optimized for performance.
// It is used when the "use_std_json" build tag is not specified.
func (aps APS) MarshalJSONFast() ([]byte, error) {
	ptr := apsPool.Get().(*[]byte)
	b := (*ptr)[:0]
	defer func() {
		*ptr = b
		apsPool.Put(ptr)
	}()

	b = append(b, '{')
	first := true

	appendQuote := func(val string) {
		b = append(b, '"')
		for i := 0; i < len(val); i++ {
			c := val[i]
			switch {
			case c == '"' || c == '\\':
				b = append(b, '\\', c)
			case c <= 0x1F:
				b = append(b, '\\', 'u', '0', '0')

				b = append(b, hex[c>>4], hex[c&0xF])
			default:
				b = append(b, c)
			}
		}
		b = append(b, '"')
	}
	addComma := func() {
		if !first {
			b = append(b, ',')
		} else {
			first = false
		}
	}

	// Alert
	if aps.Alert != nil {
		addComma()
		b = append(b, `"alert":`...)
		switch v := aps.Alert.(type) {
		case *Alert:
			tmp, _ := v.MarshalJSONFast()
			b = append(b, tmp...)
		case Alert:
			tmp, _ := v.MarshalJSONFast()
			b = append(b, tmp...)
		case string:
			appendQuote(v)
		default:
			return nil, ErrInvalidType
		}
	}

	// Badge
	if aps.Badge != nil {
		addComma()
		b = append(b, `"badge":`...)
		switch v := aps.Badge.(type) {
		case int:
			b = strconv.AppendInt(b, int64(v), 10)
		default:
			return nil, ErrInvalidType
		}
	}

	// Sound
	if aps.Sound != nil {
		addComma()
		b = append(b, `"sound":`...)
		switch v := aps.Sound.(type) {
		case *Sound:
			tmp, _ := v.MarshalJSONFast()
			b = append(b, tmp...)
		case Sound:
			tmp, _ := v.MarshalJSONFast()
			b = append(b, tmp...)
		case string:
			appendQuote(v)
		default:
			return nil, ErrInvalidType
		}
	}

	// ContentAvailable
	if aps.ContentAvailable != nil {
		addComma()
		b = append(b, `"content-available":1`...)
	}

	// MutableContent
	if aps.MutableContent != nil {
		addComma()
		b = append(b, `"mutable-content":1`...)
	}

	// Category
	if aps.Category != "" {
		addComma()
		b = append(b, `"category":`...)
		appendQuote(aps.Category)
	}

	// ThreadID
	if aps.ThreadID != "" {
		addComma()
		b = append(b, `"thread-id":`...)
		appendQuote(aps.ThreadID)
	}

	// InterruptionLevel
	if aps.InterruptionLevel != "" {
		addComma()
		b = append(b, `"interruption-level":`...)
		appendQuote(string(aps.InterruptionLevel))
	}

	// RelevanceScore
	if aps.RelevanceScore != nil {
		addComma()
		b = append(b, `"relevance-score":`...)
		switch v := aps.RelevanceScore.(type) {
		case float64:
			b = strconv.AppendFloat(b, v, 'f', -1, 64)
		default:
			return nil, ErrInvalidType
		}
	}

	// StaleDate
	if aps.StaleDate != nil {
		addComma()
		b = append(b, `"stale-date":`...)
		v := *aps.StaleDate
		b = strconv.AppendInt(b, int64(v), 10)
	}

	// FilterCriteria
	if len(aps.FilterCriteria) > 0 {
		addComma()
		b = append(b, `"filter-criteria":`...)
		appendQuote(aps.FilterCriteria)
	}

	// Timestamp
	if aps.Timestamp != nil {
		addComma()
		b = append(b, `"timestamp":`...)
		v := *aps.Timestamp
		b = strconv.AppendInt(b, int64(v), 10)

	}

	// TargetContentID
	if aps.TargetContentID != "" {
		addComma()
		b = append(b, `"target-content-id":`...)
		appendQuote(aps.TargetContentID)
	}

	// ContentState
	if len(aps.ContentState) > 0 {
		addComma()
		b = append(b, `"content-state":{`...)
		firstMap := true
		for k, v := range aps.ContentState {
			if !firstMap {
				b = append(b, ',')
			} else {
				firstMap = false
			}
			appendQuote(k)
			b = append(b, ':')
			var err error
			b, err = EncodeValue(b, v)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, '}')
	}

	// Event
	if aps.Event != "" {
		addComma()
		b = append(b, `"event":`...)
		appendQuote(aps.Event)
	}

	// DismissalDate
	if aps.DismissalDate != 0 {
		addComma()
		b = append(b, `"dismissal-date":`...)
		b = strconv.AppendInt(b, aps.DismissalDate, 10)
	}

	// AttributesType
	if aps.AttributesType != "" {
		addComma()
		b = append(b, `"attributes-type":`...)
		appendQuote(aps.AttributesType)
	}

	// Attributes
	if len(aps.Attributes) > 0 {
		addComma()
		b = append(b, `"attributes":{`...)
		firstMap := true
		for k, v := range aps.Attributes {
			if !firstMap {
				b = append(b, ',')
			} else {
				firstMap = false
			}
			appendQuote(k)
			b = append(b, ':')
			var err error
			b, err = EncodeValue(b, v)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, '}')
	}

	b = append(b, '}')
	return b, nil
}

// EncodeValue is a helper function that recursively encodes a value into a JSON byte slice.
// It supports basic types (string, int, float, bool), as well as nested maps and slices.
func EncodeValue(b []byte, v any) ([]byte, error) {
	switch val := v.(type) {
	case string:
		b = strconv.AppendQuote(b, val)
	case int:
		b = strconv.AppendInt(b, int64(val), 10)
	case int64:
		b = strconv.AppendInt(b, val, 10)
	case float64:
		b = strconv.AppendFloat(b, val, 'f', -1, 64)
	case bool:
		if val {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case nil:
		b = append(b, "null"...)
	case []byte:
		b = strconv.AppendQuote(b, string(val))
	case notification.EpochTime:
		b = strconv.AppendInt(b, int64(val), 10)
	case *notification.EpochTime:
		b = strconv.AppendInt(b, int64(*val), 10)
	case []string:
		b = append(b, '[')
		for i, v2 := range val {
			if i > 0 {
				b = append(b, ',')
			}
			b = strconv.AppendQuote(b, v2)
		}
		b = append(b, ']')
	case []int:
		b = append(b, '[')
		for i, v2 := range val {
			if i > 0 {
				b = append(b, ',')
			}
			b = strconv.AppendInt(b, int64(v2), 10)
		}
		b = append(b, ']')
	case []int64:
		b = append(b, '[')
		for i, v2 := range val {
			if i > 0 {
				b = append(b, ',')
			}
			b = strconv.AppendInt(b, v2, 10)
		}
		b = append(b, ']')
	case []float64:
		b = append(b, '[')
		for i, v2 := range val {
			if i > 0 {
				b = append(b, ',')
			}
			b = strconv.AppendFloat(b, v2, 'f', -1, 64)
		}
		b = append(b, ']')
	case json.Marshaler:
		marshaled, err := val.MarshalJSON()
		if err != nil {
			return nil, err
		}
		b = append(b, marshaled...)
	case map[string]any:
		b = append(b, '{')
		first := true
		for k2, v2 := range val {
			if !first {
				b = append(b, ',')
			} else {
				first = false
			}
			b = strconv.AppendQuote(b, k2)
			b = append(b, ':')
			var err error
			b, err = EncodeValue(b, v2)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, '}')
	case []any:
		b = append(b, '[')
		for i, v2 := range val {
			if i > 0 {
				b = append(b, ',')
			}
			var err error
			b, err = EncodeValue(b, v2)
			if err != nil {
				return nil, err
			}
		}
		b = append(b, ']')
	default:
		return nil, ErrInvalidType
	}
	return b, nil
}
