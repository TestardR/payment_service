package system

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTimeProvider(t *testing.T) {
	t.Parallel()
	provider := NewTimeProvider()

	_ = provider
}

func TestTimeProvider_Now(t *testing.T) {
	t.Parallel()
	provider := NewTimeProvider()

	before := time.Now()
	result := provider.Now()
	after := time.Now()

	assert.False(t, result.Before(before), "time should not be before the start time")
	assert.False(t, result.After(after), "time should not be after the end time")
}

func TestTimeProvider_Now_ReturnsCurrentTime(t *testing.T) {
	t.Parallel()
	provider := NewTimeProvider()

	time1 := provider.Now()
	time.Sleep(1 * time.Millisecond)
	time2 := provider.Now()

	assert.True(t, time2.After(time1), "second call to Now() should return later time, got %v then %v", time1, time2)
}
