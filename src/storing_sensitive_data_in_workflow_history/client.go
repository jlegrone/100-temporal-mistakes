package storing_sensitive_data_in_workflow_history

// @@@SNIPSTART storing-sensitive-data-good

// Good: pass a reference, not the data
type ProcessPaymentInputGood struct {
	PaymentTokenID string // Reference to payment details stored in a vault
	OrderID        string
	Amount         int64
}

// @@@SNIPEND

// @@@SNIPSTART storing-sensitive-data-bad

// Bad: pass the sensitive data directly
type ProcessPaymentInputBad struct {
	CreditCardNumber string
	CVV              string
	OrderID          string
	Amount           int64
}

// @@@SNIPEND
