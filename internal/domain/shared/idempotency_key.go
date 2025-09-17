package shared

import (
	"regexp"
)

type IdempotencyKey struct {
	value string
}

var idempotencyKeyRegex = regexp.MustCompile(`^[A-Za-z0-9]{10}$`)

func NewIdempotencyKey(value string) (IdempotencyKey, error) {
	if !idempotencyKeyRegex.MatchString(value) {
		return IdempotencyKey{}, ErrInvalidIdempotencyKey
	}

	return IdempotencyKey{value: value}, nil
}

func (k IdempotencyKey) Value() string {
	return k.value
}

func (k IdempotencyKey) String() string {
	return k.value
}

func (k IdempotencyKey) Equals(other IdempotencyKey) bool {
	return k.value == other.value
}
