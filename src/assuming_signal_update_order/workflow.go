package assuming_signal_update_order

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// Change represents an update payload with a ULID for ordering.
type Change struct {
	ID   string // ULID
	Data string
}

// @@@SNIPSTART assuming-signal-update-order-workflow-v1

// MyWorkflowV1 accepts all updates regardless of order.
// If updates arrive out of order, the workflow silently applies them
// in whatever order they were delivered.
func MyWorkflowV1(ctx workflow.Context) (string, error) {
	var lastData string

	err := workflow.SetUpdateHandler(ctx, "apply-change",
		func(ctx workflow.Context, c Change) error {
			lastData = c.Data
			return nil
		},
	)
	if err != nil {
		return "", err
	}

	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)

	return lastData, nil
}

// @@@SNIPEND

// @@@SNIPSTART assuming-signal-update-order-workflow-v2

// MyWorkflowV2 rejects out-of-order updates using a ULID-based validator.
func MyWorkflowV2(ctx workflow.Context) (string, error) {
	var lastID string
	var lastData string

	err := workflow.SetUpdateHandlerWithOptions(ctx, "apply-change",
		func(ctx workflow.Context, c Change) error {
			lastID = c.ID
			lastData = c.Data
			return nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, c Change) error {
				if c.ID <= lastID {
					return fmt.Errorf("out-of-order update: %s <= %s", c.ID, lastID)
				}
				return nil
			},
		},
	)
	if err != nil {
		return "", err
	}

	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)

	return lastData, nil
}

// @@@SNIPEND
