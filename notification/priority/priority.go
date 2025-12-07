package priority

import "strconv"

// Priority represents the APNs notification delivery priority.
type Priority int

const (
	// None omits the 'apns-priority' header, causing APNs to use the default priority of 10.
	None Priority = 0
	// PowerOnly sends the notification only when the device has power. It does not wake the device.
	PowerOnly Priority = 1
	// Conserve sends the notification with power considerations and may be delayed on low-power devices.
	Conserve Priority = 5
	// Immediate sends the notification immediately and wakes the device if necessary.
	Immediate Priority = 10
)

// String returns the string representation of the priority value.
// It returns an empty string if the priority is None (0), which signals to omit the header.
func (p Priority) String() string {
	switch p {
	case PowerOnly, Conserve, Immediate:
		return strconv.FormatInt(int64(p), 10)
	default:
		return ""
	}
}
