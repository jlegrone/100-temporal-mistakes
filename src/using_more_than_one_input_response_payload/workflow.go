package using_more_than_one_input_response_payload

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// MyActivityInput is the input type for the activity.
type MyActivityInput struct{}

// @@@SNIPSTART using-more-than-one-input-response-payload-workflow

// Avoid this
func MyWorkflowV1(ctx workflow.Context, userID string, amount int, currency string) error {
	// ...
	_ = userID
	_ = amount
	_ = currency
	return nil
}

// Prefer this
type MyWorkflowInput struct {
	UserID   string
	Amount   int
	Currency string
}

func MyWorkflowV2(ctx workflow.Context, input MyWorkflowInput) error {
	// ...
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART using-more-than-one-input-response-payload-activity

// Avoid this
func MyActivityV1(ctx context.Context, input MyActivityInput) (string, int, error) {
	// ...
	return "", 0, nil
}

// Prefer this
type MyActivityOutput struct {
	Status string
	Count  int
}

func MyActivityV2(ctx context.Context, input MyActivityInput) (MyActivityOutput, error) {
	// ...
	return MyActivityOutput{}, nil
}

// @@@SNIPEND
