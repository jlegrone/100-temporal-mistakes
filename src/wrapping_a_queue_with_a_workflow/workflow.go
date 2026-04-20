package wrapping_a_queue_with_a_workflow

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// Task represents a unit of work.
type Task struct{}

// @@@SNIPSTART wrapping-a-queue-with-a-workflow-fanout

func StartTask(ctx context.Context, temporalClient client.Client, taskID string, task Task) error {
	_, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf("task-%s", taskID),
	}, ProcessTaskWorkflow, task)
	return err
}

// @@@SNIPEND

// ProcessTaskWorkflow is a stub workflow.
func ProcessTaskWorkflow(_ workflow.Context, _ Task) error { return nil }
