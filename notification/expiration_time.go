// package notification provides types related to the metadata of an APNs notification.
package notification

import (
	"strconv"
	"time"
)

// ExpirationOnce is a special expiration value (epoch time 0) that tells APNs
// not to store the notification at all. APNs will make one attempt to deliver
// the notification, and if it cannot be delivered immediately, it will be discarded.
var ExpirationOnce = NewEpochTime(time.Time{})

// EpochTime represents a UNIX timestamp as an int64.
type EpochTime int64

// NewEpochTime creates a new EpochTime from a time.Time object.
// It returns a pointer to the EpochTime value.
func NewEpochTime(t time.Time) *EpochTime {
	if t.IsZero() {
		v := EpochTime(0)
		return &v
	}
	v := EpochTime(t.UTC().Unix())
	return &v
}

// String returns the string representation of the UNIX timestamp.
func (e EpochTime) String() string {
	return strconv.FormatInt(int64(e), 10)
}
