package assuming_signal_update_order

import (
	"fmt"

	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART assuming-signal-update-order-workflow
// Change represents an update payload with a ULID for ordering.
type Change struct {
	ID   string // ULID
	Data string
}

func MyWorkflow(ctx workflow.Context) error {
	var lastID string
	var applied []Change

	err := workflow.SetUpdateHandlerWithOptions(ctx, "apply-change",
		func(ctx workflow.Context, c Change) error {
			lastID = c.ID
			applied = append(applied, c)
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
		return err
	}

	// Wait for a "done" signal to complete the workflow.
	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)
	return nil
}

// @@@SNIPEND
