// package payload provides types for constructing the payload of an APNs notification.
package payload

import (
	"fmt"

	"github.com/takimoto3/apns/payload/sound"
)

// Sound represents the `sound` dictionary used for configuring notification sounds,
// particularly for critical alerts. The `aps` dictionary can contain either a simple
// string (the filename of the sound) or a `Sound` dictionary object.
//
// For more details, see the Apple Developer Documentation:
// https://developer.apple.com/documentation/usernotifications/generating-a-remote-notification
type Sound struct {
	// Name is the name of the sound file to be played.
	// The file should be in the app's bundle.
	Name string `json:"name,omitempty"`

	// Critical indicates whether the sound is for a critical alert.
	// Set to 1 (sound.Critical) to enable. A value of 0 or omission indicates a regular notification.
	Critical sound.AlertFlag `json:"critical,omitempty"`

	// Volume is the volume of the sound, specified as a float between 0.0 and 1.0.
	// This property is only used for critical alerts.
	Volume Ratio `json:"volume,omitempty"`
}

// Validate checks if the values of the Sound fields are valid.
// It ensures that the Critical flag is either 0 or 1, and that the Volume is within
// the valid range [0.0, 1.0].
func (s *Sound) Validate() error {
	if s.Critical != sound.None && s.Critical != sound.Critical {
		return fmt.Errorf("invalid critical flag: %d", s.Critical)
	}
	if err := s.Volume.Validate(); err != nil {
		return fmt.Errorf("volume field error: %w", err)
	}
	return nil
}
