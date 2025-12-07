// package payload provides types for constructing the payload of an APNs notification.
package payload

import (
	"fmt"
)

// Ratio represents a value between 0.0 and 1.0.
type Ratio float64

// Validate checks if the ratio is within the valid range [0.0, 1.0].
func (r Ratio) Validate() error {
	if r < 0.0 || r > 1.0 {
		return fmt.Errorf("ratio out of range: %f", r)
	}
	return nil
}
