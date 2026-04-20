package not_leveraging_workflow_input_response_payloads

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART not-leveraging-payloads-bad

// Avoid: signal-driven initialization
func OrderWorkflowBad(ctx workflow.Context) error {
	var orderID string
	ch := workflow.GetSignalChannel(ctx, "order-id")
	ch.Receive(ctx, &orderID)
	_ = orderID
	// ...
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART not-leveraging-payloads-good

// Prefer: typed input and output
type OrderInput struct {
	OrderID string
}
type OrderOutput struct {
	Status      string
	CompletedAt time.Time
}

func OrderWorkflowGood(ctx workflow.Context, input OrderInput) (OrderOutput, error) {
	// Proceed directly with input.OrderID
	return OrderOutput{
		Status:      "completed",
		CompletedAt: workflow.Now(ctx),
	}, nil
}

// @@@SNIPEND
