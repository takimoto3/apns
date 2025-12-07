package sound

// AlertFlag represents the flag for a critical alert.
type AlertFlag int

const (
	// None indicates that the alert is not critical.
	None AlertFlag = 0
	// Critical indicates that the alert is critical.
	Critical AlertFlag = 1
)
