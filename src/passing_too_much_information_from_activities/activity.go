package passing_too_much_information_from_activities

import "context"

// Customer represents a full customer record from the database.
type Customer struct {
	ID               string
	SubscriptionTier string
	IsActive         bool
	// ... many more fields
}

// db is a stub for the database layer.
var db interface {
	GetCustomer(ctx context.Context, customerID string) (*Customer, error)
}

// @@@SNIPSTART passing-too-much-information-from-activities-bad

// Bad: returning the full record
func LookupCustomerBad(ctx context.Context, customerID string) (*Customer, error) {
	return db.GetCustomer(ctx, customerID)
}

// @@@SNIPEND

// @@@SNIPSTART passing-too-much-information-from-activities-good

// Good: returning only what the workflow needs
type CustomerSummary struct {
	ID               string
	SubscriptionTier string
	IsActive         bool
}

func LookupCustomerGood(ctx context.Context, customerID string) (*CustomerSummary, error) {
	customer, err := db.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return &CustomerSummary{
		ID:               customer.ID,
		SubscriptionTier: customer.SubscriptionTier,
		IsActive:         customer.IsActive,
	}, nil
}

// @@@SNIPEND
