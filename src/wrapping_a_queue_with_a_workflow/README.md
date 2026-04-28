# Implementing a Queue with a Workflow
<!-- TODO: Rename this file to match the new title. -->

> [!TIP]
> Using a single workflow as a message queue (receiving [signals](terms/signals.md) as "messages") may create a scalability bottleneck. Temporal scales horizontally across many workflows, not vertically within one.

A tempting pattern is using a long-running workflow as a message queue: external systems send signals and the workflow processes them one by one. On the surface this gives you durability and retries for free, but it breaks down under real load.

All updates to a single workflow's [history](terms/event-history.md) are serialized through a per-workflow lock. High signal throughput causes [lock contention](workflow-lock-contention-due-to-concurrent-updates.md) and `busy_workflow` errors. Each signal adds events, pushing toward the [history length limit](overflowing-workflow-history-length.md). If the [worker](terms/worker.md) restarts, the entire history must be [replayed](terms/replay.md).

Instead, fan out to individual workflows per message (or per batch), spreading work across many histories and shards:

<!--SNIPSTART wrapping-a-queue-with-a-workflow-fanout-->
[wrapping_a_queue_with_a_workflow/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/wrapping_a_queue_with_a_workflow/workflow.go)
```go

func StartTask(ctx context.Context, temporalClient client.Client, taskID string, task Task) error {
	_, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf("task-%s", taskID),
	}, ProcessTaskWorkflow, task)
	return err
}

```
<!--SNIPEND-->

If you need some workflow-level state, partition across multiple workflows (e.g., by tenant ID or hash of message key). If you genuinely need message queue semantics (ordering, consumer groups, backpressure), use a purpose-built system (Kafka, SQS, RabbitMQ) and have workflows consume from it via activities.

<!-- TODO: add another entry for wrapping queues/jobs with workflows add to queue (eg. activity 1, result activity two). Exceptions to this would be external systems that enforce fairness, resource based scheduling, etc. But temporal is making progress here too! -->
