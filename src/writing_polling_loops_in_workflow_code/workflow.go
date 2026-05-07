package writing_polling_loops_in_workflow_code

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// Status represents the status received from a signal.
type Status struct{}

// Result represents the polling result.
type Result struct {
	Ready  bool
	Status string
}

// @@@SNIPSTART writing-polling-loops-in-workflow-code-signal

// WaitForStatusUpdate waits for a signal instead of polling,
// adding zero events to the history while waiting.
func WaitForStatusUpdate(ctx workflow.Context) (Status, error) {
	var status Status
	signalChan := workflow.GetSignalChannel(ctx, "status-update")
	signalChan.Receive(ctx, &status)
	return status, nil
}

// @@@SNIPEND

// @@@SNIPSTART writing-polling-loops-in-workflow-code-activity

// PollUntilReady polls an external system inside an activity with
// heartbeats. The activity can poll as frequently as needed without
// adding events to workflow history.
func PollUntilReady(ctx context.Context) (Result, error) {
	for {
		result, err := checkExternalSystem()
		if err != nil {
			return Result{}, err
		}
		if result.Ready {
			return result, nil
		}
		activity.RecordHeartbeat(ctx, result.Status)
		time.Sleep(30 * time.Second) // Regular time.Sleep, not workflow.Sleep
	}
}

// @@@SNIPEND

// checkExternalSystem is a stub for polling an external system.
func checkExternalSystem() (Result, error) {
	return Result{Ready: true}, nil
}
