# Exceeding the 10-Second Workflow Task Timeout

> [!TIP]
> * Workflow tasks have a default timeout of 10 seconds. If a workflow task takes longer than that, it times out and is retried, potentially causing a livelock.
> * Common causes: expensive computation in workflow code, replaying a large history, or scheduling too many operations in a single workflow task.
> * Solution: keep workflow code lightweight, use [ContinueAsNew](terms/continue-as-new.md) to bound history size, and delegate heavy work to activities.

## What?

A [workflow task](terms/workflow-task.md) is the unit of work that the Temporal [worker](terms/worker.md) processes when executing workflow code. Each time the worker picks up a workflow task, it [replays](terms/replay.md) the [workflow history](terms/event-history.md) and then executes new workflow code until the next yield point (e.g., waiting for an activity, a timer, or a [signal](terms/signals.md)). By default, the server expects a workflow task to complete within 10 seconds.

If the worker fails to return a result within that window, the server considers the task timed out and reschedules it. The worker picks it up again, replays the same history, hits the same bottleneck, times out again -- and the cycle repeats. The workflow is stuck in a livelock: it is not [terminated](terms/terminate.md), but it cannot make forward progress either.

You will see this manifest as `WorkflowTaskTimedOut` events accumulating in the workflow history and `workflow_task_schedule_to_start_latency` / `workflow_task_execution_latency` metrics spiking.

## Why?

Three common scenarios lead to exceeding the workflow task timeout:

1. **Expensive computation in workflow code.** Workflow code runs on the worker's workflow task processing goroutine (or thread). CPU-intensive operations like JSON parsing of large [payloads](terms/payload.md), complex data transformations, or cryptographic operations eat into the 10-second budget. Unlike activities, workflow code has no built-in mechanism for [heartbeating](terms/heartbeat.md) or extending the deadline.

2. **Replaying a large history.** Every workflow task starts by replaying the full event history (unless the workflow is cached in memory). A workflow with tens of thousands of events can spend most of the 10-second budget just on [replay](terms/replay.md), leaving little time for executing new code. This problem compounds: the more the workflow grows, the closer each task gets to the timeout, until eventually replay alone exceeds 10 seconds and the workflow is permanently stuck.

3. **Scheduling too many operations in a single task.** If your workflow code loops and schedules thousands of activities or [child workflows](terms/child-workflow.md) without yielding, the resulting workflow task completion message can become very large and slow to process, pushing the task past the timeout.

## Solution

1. **Keep workflow code lightweight.** Workflow functions should only make decisions and orchestrate work. Move any computation that could be expensive into activities, where you have proper timeout control, retries, and heartbeating. A good rule of thumb: if you cannot predict that a piece of code will complete in under a second, it belongs in an activity.

2. **Use [ContinueAsNew](terms/continue-as-new.md) to bound history size.** By periodically continuing as new, you keep the [history event count](<overflowing-workflow-history-size.md>) small, which directly reduces replay time. This is especially critical for long-running workflows that accumulate events over days or weeks.

3. **Batch large fan-outs.** If you need to schedule thousands of activities, do it in batches rather than all at once. Schedule a batch, yield (e.g., by awaiting the batch results or a short timer), then schedule the next batch. This keeps individual workflow task sizes manageable.

4. **Increase the timeout as a last resort.** The workflow task timeout is configurable per workflow via `WorkflowTaskTimeout` in the workflow options. You can increase it beyond 10 seconds, but treat this as a band-aid. A workflow task that routinely needs more than 10 seconds is a sign of a deeper design issue that will only get worse as the workflow history grows.

5. **Ensure workflow caching is effective.** When workflows remain cached in memory on the worker, subsequent workflow tasks skip replay entirely and execute only the new events. Make sure your worker's cache size (`WorkerStickyTaskQueueScheduleToStartTimeout` and cache configuration) is tuned appropriately for your workload. Cache evictions force full replays, which are the most common trigger for timeout issues.
