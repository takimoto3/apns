package interruptionlevel

// InterruptionLevel represents the notification's importance level.
type InterruptionLevel string

const (
	// Active brings the notification to the forefront.
	Active InterruptionLevel = "active"
	// Passive adds the notification to the notification list without interrupting the user.
	Passive InterruptionLevel = "passive"
	// TimeSensitive presents the notification immediately.
	TimeSensitive InterruptionLevel = "time-sensitive"
	// Critical presents the notification immediately and may bypass Do Not Disturb.
	Critical InterruptionLevel = "critical"
)
