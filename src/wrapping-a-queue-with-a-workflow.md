# Wrapping a Queue with a Workflow

> [!TIP]
> * Using a single workflow as a message queue (receiving signals as "messages") creates a scalability bottleneck.
> * A single workflow's [history](terms/event-history.md) shard becomes a hot spot under high throughput, leading to [lock contention](workflow-lock-contention-due-to-concurrent-updates.md) and growing history.
> * Spread work across multiple workflows or use dedicated queuing systems for queue semantics.

## What?

A tempting pattern when building on Temporal is to use a single long-running workflow as a message queue: external systems send [signals](terms/signals.md) as "messages" and the workflow processes them one by one (or in batches) from its signal channel.

```go
func QueueWorkflow(ctx workflow.Context) error {
    ch := workflow.GetSignalChannel(ctx, "tasks")
    for {
        var task Task
        ch.Receive(ctx, &task)
        // Process each task sequentially
        err := workflow.ExecuteActivity(ctx, ProcessTask, task).Get(ctx, nil)
        if err != nil {
            // handle error
        }
    }
}
```

On the surface this looks elegant: you get durability, retries, and visibility for free. In practice, it breaks down quickly under any real load.

## Why?

All updates to a single workflow's history are serialized through a [workflow-level lock](workflow-lock-contention-due-to-concurrent-updates.md). When many signals arrive concurrently, they all compete for this lock. The result is:

1. **Lock contention**: As signal throughput increases, you'll see a rise in `busy_workflow` errors and dramatically increased end-to-end latency. The workflow becomes a bottleneck.
2. **Unbounded history growth**: Each signal adds events to the workflow history. A workflow receiving hundreds or thousands of messages will quickly approach the history size limit (50k events by default), at which point the server [terminates](terms/terminate.md) the workflow.
3. **Replay cost**: If the [worker](terms/worker.md) restarts or the workflow gets evicted from cache, the entire history must be [replayed](terms/replay.md). A workflow with thousands of signal events will take a long time to replay.
4. **Single point of failure**: All your "queue processing" is concentrated in one workflow on one shard. If that shard has issues, everything stops.

Temporal scales horizontally across many workflows. It does not scale vertically within a single workflow. Using a workflow as a queue fights against the fundamental architecture.

## How?

There are several alternatives depending on your needs:

### Fan out to individual workflows

Instead of funneling messages into one workflow, start a separate workflow per message (or per batch):

```go
// In your signal sender / API handler
_, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
    ID: fmt.Sprintf("task-%s", taskID),
}, ProcessTaskWorkflow, task)
```

This spreads work across many workflow histories and shards, leveraging Temporal's horizontal scalability.

### Use task queues directly

If the "messages" are really units of work to be processed, consider modeling them as activities dispatched through Temporal's built-in [task queue](terms/task-queue.md) mechanism rather than routing them through a workflow's signal channel.

### Use a dedicated message queue

If you genuinely need message queue semantics (ordering guarantees, consumer groups, backpressure), use a system designed for that purpose (Kafka, SQS, RabbitMQ, etc.) and have Temporal workflows consume from it via activities. Temporal is an orchestration engine, not a message broker.

### Partitioned workflows

If you need some workflow-level state around message processing, partition across multiple workflows (e.g., by tenant ID, by hash of message key) to spread the load:

```go
workflowID := fmt.Sprintf("processor-%d", hash(messageKey) % numPartitions)
temporalClient.SignalWorkflow(ctx, workflowID, "", "tasks", task)
```

This is still not a queue, but it reduces the hot-spot problem by distributing signals across multiple workflows.
