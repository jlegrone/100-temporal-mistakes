package assuming_workflow_timeouts_allow_graceful_cleanup

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART assuming-workflow-timeouts-bad

// MyWorkflowV1 relies on the workflow execution timeout for its
// business deadline. When the timeout fires, the workflow is
// terminated -- no cleanup runs.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	ch := workflow.GetSignalChannel(ctx, "done")
	ch.Receive(ctx, nil)

	log.Info("done")
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART assuming-workflow-timeouts-good

// MyWorkflowV2 uses an internal timer as the business deadline.
// If the timer fires before the signal arrives, the workflow
// completes gracefully with an error instead of being terminated.
func MyWorkflowV2(ctx workflow.Context, deadline time.Duration) error {
	log := workflow.GetLogger(ctx)

	ch := workflow.GetSignalChannel(ctx, "done")
	timer := workflow.NewTimer(ctx, deadline)

	selector := workflow.NewSelector(ctx)

	var timedOut bool
	selector.AddReceive(ch, func(c workflow.ReceiveChannel, more bool) {
		log.Info("done")
	})
	selector.AddFuture(timer, func(f workflow.Future) {
		log.Warn("deadline exceeded", "deadline", deadline)
		timedOut = true
	})
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("workflow canceled")
	})
	selector.Select(ctx)

	if ctx.Err() != nil {
		return ctx.Err()
	}
	if timedOut {
		return workflow.NewContinueAsNewError(ctx, MyWorkflowV2, deadline)
	}
	return nil
}

// @@@SNIPEND
