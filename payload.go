// package apns provides a client for sending notifications to the Apple Push Notification service.
package apns

import (
	"encoding/json"
	"maps"

	"github.com/takimoto3/apns/payload"
)

// Payload represents the JSON payload of an APNs notification.
// It consists of the standard `aps` dictionary and any custom data.
//
// For more details, see the Apple Developer Documentation:
// https://developer.apple.com/documentation/usernotifications/generating-a-remote-notification
type Payload struct {
	// APS is the Apple-defined dictionary that contains notification-specific data.
	APS payload.APS `json:"aps"`

	// CustomData is a map for any app-specific custom data.
	// The keys and values in this map will be merged at the root level of the
	// JSON payload, alongside the `aps` dictionary.
	CustomData map[string]any `json:",inline"`
}

// MarshalJSON implements the `json.Marshaler` interface.
// It customizes the JSON output by merging the `APS` dictionary and the `CustomData`
// map at the root level of the payload. This is necessary because the `json:",inline"`
// struct tag does not work as expected with an embedded struct.
func (p *Payload) MarshalJSON() ([]byte, error) {
	if len(p.CustomData) == 0 {
		// If there is no custom data, just marshal the APS dictionary.
		return json.Marshal(map[string]any{"aps": p.APS})
	}

	// If there is custom data, merge it with the APS dictionary.
	mp := maps.Clone(p.CustomData)
	mp["aps"] = p.APS
	return json.Marshal(mp)
}
