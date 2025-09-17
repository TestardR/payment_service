package shared

import (
	"fmt"
	"math"
)

type Amount struct {
	value int64 // Store as cents to avoid floating point issues
}

func NewAmount(value float64) (Amount, error) {
	if value < 0 {
		return Amount{}, ErrInvalidAmount
	}

	if value > math.MaxInt64/100 {
		return Amount{}, ErrInvalidAmount
	}

	// Convert to cents and round to avoid floating point precision issues
	cents := int64(math.Round(value * 100))

	return Amount{value: cents}, nil
}

func NewAmountFromCents(cents int64) (Amount, error) {
	if cents < 0 {
		return Amount{}, ErrInvalidAmount
	}

	return Amount{value: cents}, nil
}

func (a Amount) Value() float64 {
	return float64(a.value) / 100
}

func (a Amount) Cents() int64 {
	return a.value
}

func (a Amount) String() string {
	return fmt.Sprintf("%.2f", a.Value())
}

func (a Amount) Equals(other Amount) bool {
	return a.value == other.value
}

func (a Amount) IsZero() bool {
	return a.value == 0
}

func (a Amount) Add(other Amount) Amount {
	return Amount{value: a.value + other.value}
}

func (a Amount) Subtract(other Amount) (Amount, error) {
	if a.value < other.value {
		return Amount{}, fmt.Errorf("cannot subtract, result would be negative")
	}
	return Amount{value: a.value - other.value}, nil
}
