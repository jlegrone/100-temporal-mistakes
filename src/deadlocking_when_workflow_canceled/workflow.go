package deadlocking_when_workflow_canceled

import (
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART deadlocking-canceled-bad

// MyWorkflowV1 blocks forever if canceled. The Receive call blocks
// until a signal arrives, but once the workflow is canceled, no signal
// will ever be delivered.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	log.Debug("waiting for done signal")

	ch := workflow.GetSignalChannel(ctx, "done")

	ch.Receive(ctx, nil)
	log.Debug("received done signal")

	return nil
}

// @@@SNIPEND

// @@@SNIPSTART deadlocking-canceled-good

// MyWorkflowV2 uses a selector to unblock on either the signal
// or cancellation.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)
	log.Debug("waiting for done signal")

	ch := workflow.GetSignalChannel(ctx, "done")
	selector := workflow.NewSelector(ctx)

	selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
		log.Debug("received done signal")
	})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("received done", "error", ctx.Err())
	})
	selector.Select(ctx)
	return ctx.Err()
}

// @@@SNIPEND
