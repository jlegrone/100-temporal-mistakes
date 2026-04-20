# Not Draining Activity Tasks Before Shutdown

> [!TIP]
> When a [worker](../terms/worker.md) is killed during deployment, in-flight activities are abandoned. Configure a graceful shutdown drain period so activities can finish or checkpoint via [heartbeats](../terms/heartbeat.md).

During deployments, workers are stopped and replaced. Without a drain period, the server doesn't know activities failed until their timeout expires, then retries from scratch -- wasting all completed work. This is especially painful for long-running activities where losing minutes of progress per deployment adds up.

<!--SNIPSTART not-draining-activity-tasks-example-->
[not_draining_activity_tasks_before_shutdown/examples.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_draining_activity_tasks_before_shutdown/examples.go)
```go
func newWorkerWithGracefulStop(c client.Client) worker.Worker {
	return worker.New(c, "my-task-queue", worker.Options{
		WorkerStopTimeout: 5 * time.Minute,
	})
}

```
<!--SNIPEND-->

Make sure your deployment orchestrator (Kubernetes `terminationGracePeriodSeconds`, ECS stop timeout) gives the worker at least as much time as the drain period. Activities that [heartbeat](../terms/heartbeat.md) detect [cancellation](../terms/cancelation.md) (triggered by shutdown) and save progress, so they resume from the last checkpoint on retry rather than starting over.

Match the drain timeout to your longest reasonable activity duration. Activities that run for hours should already be heartbeating and checkpointing, so a shorter drain is fine as long as they can exit promptly.

See also: [Not Using Activity Heartbeat Details](../not_using_activity_heartbeat_details/).
