// package apns provides a client for sending notifications to the Apple Push Notification service.
package apns

import (
	"sync"

	"github.com/takimoto3/apns/payload"
)

var customDataBufSize = 512

var customDataPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, customDataBufSize)
		return &b
	},
}

// MarshalJSONFast is a custom JSON marshaler for the Payload type that is optimized for performance.
// It is used when the "use_std_json" build tag is not specified.
func (p Payload) MarshalJSONFast() ([]byte, error) {
	var err error
	// --- 1. aps ---
	apsBytes, err := p.APS.MarshalJSONFast()
	if err != nil {
		return nil, err
	}
	// --- 2. CustomData ---
	var customDataBytes []byte
	if len(p.CustomData) > 0 {
		ptr := customDataPool.Get().(*[]byte)
		b := (*ptr)[:0]
		defer func() {
			*ptr = b
			customDataPool.Put(ptr)
		}()

		customDataBytes, err = marshalCustomData(b, p.CustomData)
		if err != nil {
			return nil, err
		}
	}

	// Estimate buffer size: len(apsBytes) + len(customDataBytes) + 12
	// 12 = { } + "aps": + comma + some extra margin
	b := make([]byte, 0, len(apsBytes)+len(customDataBytes)+12)
	b = append(b, '{')

	b = append(b, `"aps":`...)
	b = append(b, apsBytes...)

	// --- 2. CustomData ---
	if len(p.CustomData) > 0 {
		b = append(b, ',')
		b = append(b, customDataBytes...)
	}
	b = append(b, '}')

	return b, nil
}

func marshalCustomData(b []byte, data map[string]any) ([]byte, error) {
	first := true
	addComma := func() {
		if !first {
			b = append(b, ',')
		}
		first = false
	}
	appendQuote := func(val string) {
		b = append(b, '"')
		for i := 0; i < len(val); i++ {
			c := val[i]
			if c == '"' || c == '\\' {
				b = append(b, '\\', c)
			} else {
				b = append(b, c)
			}
		}
		b = append(b, '"')
	}
	// --- 2. CustomData ---
	for k, v := range data {
		addComma()
		appendQuote(k)
		b = append(b, ':')
		var err error
		b, err = payload.EncodeValue(b, v)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}
