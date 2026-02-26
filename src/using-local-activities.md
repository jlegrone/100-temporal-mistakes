# Using Local Activities

> [!TIP]
> * [Local activities](terms/local-activity.md) execute on the same [worker](terms/worker.md) without a round trip to the Temporal server, reducing latency and [history](terms/event-history.md) size.
> * The [workflow task](terms/workflow-task.md) timeout (default 10s) must be long enough to cover the local activity's execution -- if it isn't, the workflow task times out and the entire workflow task is retried from scratch.
> * Use local activities only for short, fast operations like data transformations, lightweight lookups, or validations.

## What?

Local activities are an optimization that skips the normal activity scheduling flow. Instead of the worker sending a task to the Temporal server and the server dispatching it back to a worker's [task queue](terms/task-queue.md), the local activity runs directly in the same workflow task on the same worker. The result is recorded in a single workflow task completion event rather than the usual `ActivityTaskScheduled` / `ActivityTaskCompleted` pair.

```go
func MyWorkflow(ctx workflow.Context, input Input) error {
    localCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
        ScheduleToCloseTimeout: 5 * time.Second,
    })
    var result Result
    err := workflow.ExecuteLocalActivity(localCtx, QuickLookup, input).Get(ctx, &result)
    // ...
}
```

This sounds like a pure win -- less latency, fewer history events. But local activities come with important constraints that are easy to overlook.

## Why?

Local activities have several caveats that can cause serious problems if not understood:

1. **Bound by the workflow task timeout**: A local activity runs within the context of a workflow task. The workflow task has a timeout (default 10s, see [exceeding the default task timeout](exceeding-10s-task-timeout.md)). If the local activity takes longer than the remaining workflow task time, the entire workflow task times out. The server then schedules a new workflow task, the worker [replays](terms/replay.md) the workflow, and the local activity runs again from scratch. This can create an infinite loop where the local activity never completes.

2. **No independent visibility**: Local activities don't generate their own events in the history until the workflow task completes successfully. This means you can't see them in progress in the Temporal UI, and if the workflow task fails, there's no trace of the local activity attempt.

3. **Retry behavior**: When a local activity fails and is retried, the retry happens locally on the same worker. But if the workflow task itself times out (because the local activity took too long), the entire workflow task is retried -- which means replaying the workflow from the beginning and running the local activity again. This is fundamentally different from normal activity retries.

4. **No load balancing**: Normal activities are dispatched through the task queue and can be picked up by any worker. Local activities are locked to the worker running the workflow task. If that worker is overloaded, the local activity competes for resources with other workflow tasks on the same worker.

5. **Worker restarts**: If the worker restarts while a local activity is running, there is no server-side record of the attempt. The workflow task times out and is retried on another worker.

## How?

Local activities are a good fit for:

- **Data transformations**: Converting, validating, or enriching data that doesn't require I/O.
- **Fast lookups**: Reading from a local cache or a very fast in-memory store.
- **Lightweight I/O**: Quick database reads or API calls that reliably complete in under a second.

Local activities are a bad fit for:

- **Long-running operations**: Anything that might take more than a few seconds.
- **Unreliable external calls**: API calls to services with variable latency or frequent timeouts.
- **Heavy computation**: CPU-intensive work that could starve other workflow tasks on the same worker.

As a rule of thumb, if you need to think about whether your operation will finish within the workflow task timeout, use a normal activity instead. The small latency savings of a local activity is not worth the complexity and risk of workflow task timeouts.

If you do use local activities, explicitly set the `ScheduleToCloseTimeout` to a value well below your workflow task timeout to leave room for other local activities and workflow logic within the same workflow task.
