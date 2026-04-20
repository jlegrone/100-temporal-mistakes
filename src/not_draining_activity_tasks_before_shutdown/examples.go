package not_draining_activity_tasks_before_shutdown

import (
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// @@@SNIPSTART not-draining-activity-tasks-example
func newWorkerWithGracefulStop(c client.Client) worker.Worker {
	return worker.New(c, "my-task-queue", worker.Options{
		WorkerStopTimeout: 5 * time.Minute,
	})
}

// @@@SNIPEND
