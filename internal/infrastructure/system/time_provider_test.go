package system

import (
	"testing"
	"time"
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
	if result.Before(before) || result.After(after) {
		t.Errorf("expected time to be between %v and %v, got %v", before, after, result)
	}
}

func TestTimeProvider_Now_ReturnsCurrentTime(t *testing.T) {
	provider := NewTimeProvider()
	
	// Call Now() multiple times and verify they're different (or very close)
	time1 := provider.Now()
	time.Sleep(1 * time.Millisecond) // Small delay to ensure different times
	time2 := provider.Now()
	
	if !time2.After(time1) {
		t.Errorf("expected second call to Now() to return later time, got %v then %v", time1, time2)
	}
}
