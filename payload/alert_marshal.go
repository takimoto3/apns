//go:build !use_std_json
// +build !use_std_json

// package payload provides types for constructing the payload of an APNs notification.
package payload

import (
	"sync"
)

const hex = "0123456789abcdef"

var alertBufSize = 512

var alertPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, alertBufSize)
		return &b
	},
}

// MarshalJSONFast is a custom JSON marshaler for the Alert type that is optimized
// for performance. It is used when the "use_std_json" build tag is not specified.
func (a Alert) MarshalJSONFast() ([]byte, error) {
	ptr := alertPool.Get().(*[]byte)
	b := (*ptr)[:0]
	defer func() {
		*ptr = b
		alertPool.Put(ptr)
	}()

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
	addString := func(key, val string) {
		b = append(b, '"')
		b = append(b, key...)
		b = append(b, '"', ':')
		appendQuote(val)
	}
	addStringSlice := func(key string, vals []string) {
		b = append(b, '"')
		b = append(b, key...)
		b = append(b, '"', ':', '[')
		for i, v := range vals {
			if i > 0 {
				b = append(b, ',')
			}
			appendQuote(v)
		}
		b = append(b, ']')
	}

	b = append(b, '{')
	if a.Title != "" {
		addComma()
		addString("title", a.Title)
	}
	if a.Subtitle != "" {
		addComma()
		addString("subtitle", a.Subtitle)
	}
	if a.Body != "" {
		addComma()
		addString("body", a.Body)
	}
	if a.LaunchImage != "" {
		addComma()
		addString("launch-image", a.LaunchImage)
	}
	if a.LocKey != "" {
		addComma()
		addString("loc-key", a.LocKey)
	}
	if len(a.LocArgs) > 0 {
		addComma()
		addStringSlice("loc-args", a.LocArgs)
	}

	if a.TitleLocKey != "" {
		addComma()
		addString("title-loc-key", a.TitleLocKey)
	}
	if len(a.TitleLocArgs) > 0 {
		addComma()
		addStringSlice("title-loc-args", a.TitleLocArgs)
	}

	if a.SubtitleLocKey != "" {
		addComma()
		addString("subtitle-loc-key", a.SubtitleLocKey)
	}
	if len(a.SubtitleLocArgs) > 0 {
		addComma()
		addStringSlice("subtitle-loc-args", a.SubtitleLocArgs)
	}

	if a.ActionLocKey != "" {
		addComma()
		addString("action-loc-key", a.ActionLocKey)
	}
	b = append(b, '}')

	return b, nil
}
