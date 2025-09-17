package system

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTimeProvider(t *testing.T) {
	provider := NewTimeProvider()

	// Test that it returns a value type (not pointer)
	// TimeProvider is a simple struct, so we just verify it was created
	// The real test is that NewTimeProvider() compiles and runs without error
	_ = provider // Use the provider to avoid unused variable error
}

func TestTimeProvider_Now(t *testing.T) {
	provider := NewTimeProvider()

	before := time.Now()
	result := provider.Now()
	after := time.Now()

	// Verify that the returned time is between before and after
	assert.False(t, result.Before(before), "time should not be before the start time")
	assert.False(t, result.After(after), "time should not be after the end time")
}

func TestTimeProvider_Now_ReturnsCurrentTime(t *testing.T) {
	provider := NewTimeProvider()

	// Call Now() multiple times and verify they're different (or very close)
	time1 := provider.Now()
	time.Sleep(1 * time.Millisecond) // Small delay to ensure different times
	time2 := provider.Now()

	assert.True(t, time2.After(time1), "second call to Now() should return later time, got %v then %v", time1, time2)
}
