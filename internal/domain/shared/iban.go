package shared

import (
	"regexp"
	"strings"
)

type IBAN struct {
	value string
}

var ibanRegex = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{4}[0-9]{7}([A-Z0-9]?){0,16}$`)

func NewIBAN(value string) (IBAN, error) {
	normalized := strings.ToUpper(strings.ReplaceAll(value, " ", ""))

	if !ibanRegex.MatchString(normalized) {
		return IBAN{}, ErrInvalidIBAN
	}

	return IBAN{value: normalized}, nil
}

func (i IBAN) Value() string {
	return i.value
}

func (i IBAN) String() string {
	return i.value
}

func (i IBAN) Equals(other IBAN) bool {
	return i.value == other.value
}
