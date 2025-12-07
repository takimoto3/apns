// package payload provides types for constructing the payload of an APNs notification.
package payload

import (
	"errors"
	"fmt"

	"github.com/takimoto3/apns/notification"
	"github.com/takimoto3/apns/payload/interruptionlevel"
)

// APS represents the `aps` dictionary, which is the core of an APNs payload.
// It contains system-defined keys that control how the system delivers and
// displays the notification.
//
// For more details, see the Apple Developer Documentation:
// https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns
type APS struct {
	// Alert is the content of the alert message.
	// It can be a simple string or a dictionary object (payload.Alert).
	Alert any `json:"alert,omitempty"`

	// Badge is the number to display in a badge on the app's icon.
	// Specify an integer. To remove the badge, set this to 0.
	Badge any `json:"badge,omitempty"`

	// Sound is the name of a sound file in the app's bundle or a dictionary
	// object (payload.Sound) for critical alerts.
	Sound any `json:"sound,omitempty"`

	// ContentAvailable provides a way to wake up your app in the background.
	// Set to 1 to indicate that new content is available.
	ContentAvailable any `json:"content-available,omitempty"`

	// MutableContent allows a Notification Service App Extension to modify the
	// notification's content.
	// Set to 1 to enable this feature.
	MutableContent any `json:"mutable-content,omitempty"`

	// Category is the identifier for a registered category of actionable notifications.
	Category string `json:"category,omitempty"`

	// ThreadID is an identifier to group related notifications.
	ThreadID string `json:"thread-id,omitempty"`

	// InterruptionLevel indicates the importance and delivery timing of a notification.
	InterruptionLevel interruptionlevel.InterruptionLevel `json:"interruption-level,omitempty"`

	// RelevanceScore is a value between 0.0 and 1.0 that determines the sorting order
	// of notifications in the Notification Summary. For Live Activities, the value can be > 1.0.
	RelevanceScore any `json:"relevance-score,omitempty"`

	// StaleDate is the time at which a Live Activity becomes stale.
	// The value is a UNIX epoch timestamp in seconds.
	StaleDate *notification.EpochTime `json:"stale-date,omitempty"`

	// FilterCriteria are the criteria that determine whether a notification is
	// shown in a particular Focus mode.
	FilterCriteria string `json:"filter-criteria,omitempty"`

	// Timestamp is the time you sent the remote notification that starts, updates,
	// or ends a Live Activity.
	// The value is a UNIX epoch timestamp in seconds.
	Timestamp *notification.EpochTime `json:"timestamp,omitempty"`

	// TargetContentID is the identifier of the window that will be brought forward.
	TargetContentID string `json:"target-content-id,omitempty"`

	// ContentState is the dictionary that contains the dynamic data for a Live Activity.
	ContentState map[string]any `json:"content-state,omitempty"`

	// Event is a string that describes the state of a Live Activity.
	// It can be "start", "update", or "end".
	Event string `json:"event,omitempty"`

	// DismissalDate is the time at which the system dismisses a Live Activity.
	// The value is a UNIX epoch timestamp in seconds.
	DismissalDate int64 `json:"dismissal-date,omitempty"`

	// AttributesType is the name of the custom struct that defines the static
	// properties of a Live Activity.
	AttributesType string `json:"attributes-type,omitempty"`

	// Attributes is the dictionary that contains the static data for a Live Activity.
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Validate checks the types and values of the fields in the APS dictionary.
// It ensures that fields like Alert, Badge, and Sound have compatible types,
// and that values like RelevanceScore and InterruptionLevel are within valid ranges.
func (aps *APS) Validate() error {
	isNotification :=
		aps.Alert != nil ||
			aps.Badge != nil ||
			aps.Sound != nil ||
			aps.ContentAvailable != nil ||
			aps.MutableContent != nil

	isLiveActivity :=
		len(aps.ContentState) > 0 ||
			len(aps.Attributes) > 0

	// Check if the APS dictionary is effectively empty.
	if !isNotification && !isLiveActivity {
		return errors.New("aps dictionary must not be empty")
	}

	// Validate Alert
	if aps.Alert != nil {
		switch aps.Alert.(type) {
		case string, Alert, *Alert:
			// valid types
		default:
			return fmt.Errorf("invalid type for aps.Alert: must be string, Alert, or *Alert")
		}
	}

	// Validate Badge
	if aps.Badge != nil {
		if _, ok := aps.Badge.(int); !ok {
			return fmt.Errorf("invalid type for aps.Badge: must be an integer")
		}
	}

	// Validate Sound
	if aps.Sound != nil {
		switch s := aps.Sound.(type) {
		case string:
			// valid type
		case Sound:
			if err := s.Validate(); err != nil {
				return err
			}
		case *Sound:
			if err := s.Validate(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid type for aps.Sound: must be string, Sound, or *Sound")
		}
	}

	// Validate ContentAvailable
	if aps.ContentAvailable != nil {
		v, ok := aps.ContentAvailable.(int)
		if !ok || v != 1 {
			return fmt.Errorf("invalid value for aps.ContentAvailable: must be the integer 1")
		}
	}

	// Validate MutableContent
	if aps.MutableContent != nil {
		v, ok := aps.MutableContent.(int)
		if !ok || v != 1 {
			return fmt.Errorf("invalid value for aps.MutableContent: must be the integer 1")
		}
	}

	// Validate InterruptionLevel
	if aps.InterruptionLevel != "" {
		switch aps.InterruptionLevel {
		case interruptionlevel.Passive, interruptionlevel.Active, interruptionlevel.TimeSensitive, interruptionlevel.Critical:
			// valid types
		default:
			return fmt.Errorf("invalid value for aps.InterruptionLevel: %s", aps.InterruptionLevel)
		}
	}

	// Validate Event
	if aps.Event != "" {
		// Event must be "start", "update", or "end"
		switch aps.Event {
		case "start":
		case "update":
		case "end":
		default:
			return fmt.Errorf("invalid value for aps.Event: %s", aps.Event)
		}
	}

	// Validate RelevanceScore
	if aps.RelevanceScore != nil {
		var score float64
		var ok bool
		if score, ok = aps.RelevanceScore.(float64); !ok {
			if intScore, ok := aps.RelevanceScore.(int); ok {
				score = float64(intScore) // intをfloat64に変換
			} else {
				return fmt.Errorf("invalid type for aps.RelevanceScore: must be a number (float64 or int)")
			}
		}

		if !isLiveActivity {
			if score < 0.0 || score > 1.0 {
				return fmt.Errorf("relevance-score must be between 0.0 and 1.0 for standard notifications, but got %f", score)
			}
		}
	}

	return nil
}
