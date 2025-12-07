//go:build !use_std_json
// +build !use_std_json

// package payload provides types for constructing the payload of an APNs notification.
package payload

import "strconv"

// MarshalJSONFast is a custom JSON marshaler for the Sound type that is optimized
// for performance. It is used when the "use_std_json" build tag is not specified.
func (s Sound) MarshalJSONFast() ([]byte, error) {
	b := make([]byte, 0, 64)
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
		}
		first = false
	}

	// critical
	if s.Critical != 0 {
		addComma()
		b = append(b, `"critical":`...)
		b = strconv.AppendInt(b, int64(s.Critical), 10)
	}

	// name
	if s.Name != "" {
		addComma()
		b = append(b, `"name":`...)
		appendQuote(s.Name)
	}

	// volume
	if s.Volume != 0 {
		addComma()
		b = append(b, `"volume":`...)
		b = strconv.AppendFloat(b, float64(s.Volume), 'f', -1, 64)
	}

	b = append(b, '}')

	return b, nil
}
