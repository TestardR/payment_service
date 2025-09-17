package payment

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "PENDING"
	StatusProcessed PaymentStatus = "PROCESSED"
	StatusFailed    PaymentStatus = "FAILED"
)

func (s PaymentStatus) String() string {
	return string(s)
}

func (s PaymentStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessed, StatusFailed:
		return true
	default:
		return false
	}
}

func (s PaymentStatus) IsFinal() bool {
	return s == StatusProcessed || s == StatusFailed
}
