package system

import "time"

type TimeProvider struct{}

func NewTimeProvider() TimeProvider {
	return TimeProvider{}
}

func (t TimeProvider) Now() time.Time {
	return time.Now()
}
