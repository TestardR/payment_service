package shared

import "time"

type TimeProvider interface {
	Now() time.Time
}
