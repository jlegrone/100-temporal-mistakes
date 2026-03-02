# Not Using ContinueAsNew

> [!TIP]
> * Long-running workflows accumulate events in their history, leading to longer [replay](terms/replay.md) times and eventually hitting the [history size limit](<overflowing-workflow-history-size.md>).
> * [ContinueAsNew](terms/continue-as-new.md) creates a fresh workflow execution with a new history, carrying over only the essential state.
> * It also limits the age of your code, greatly simplifying [versioning](terms/versioning.md).

## What?

Every action in a Temporal workflow -- scheduling an activity, receiving a [signal](terms/signals.md), firing a timer -- adds events to the workflow's [history](terms/event-history.md). For workflows that run indefinitely or for a long time (event listeners, polling loops, subscription managers, recurring jobs), this history grows without bound.

Without [ContinueAsNew](terms/continue-as-new.md), the workflow eventually hits the history size limit (50k events by default) and the server [terminates](terms/terminate.md) it. Even before that hard limit, large histories cause performance problems due to increasingly slow replays.

## Why?

**Replay performance**: When a [worker](terms/worker.md) picks up a [workflow task](terms/workflow-task.md) (or restarts after a crash), it replays the entire history to reconstruct the workflow's state. A history with 40,000 events takes significantly longer to replay than one with 200 events. At scale, this impacts worker throughput and workflow latency.

**History size limit**: The Temporal server terminates workflows that exceed the maximum history size (50k events by default, configurable via [dynamic configuration](terms/dynamic-config.md)). A terminated workflow gets no chance to clean up -- it stops.

**Versioning complexity**: The longer a workflow runs, the more code versions it spans. If a workflow has been running for months, you may need to maintain compatibility with code paths written months ago. [ContinueAsNew](terms/continue-as-new.md) resets the history, so the new execution starts fresh with the latest code. This dramatically simplifies [versioning](terms/versioning.md) and enables tools like the [Temporal Worker Kubernetes Controller](<terms/temporal-worker-kubernetes-controller.md>) to manage fewer concurrent versions.

**Memory pressure**: Workers keep cached workflow state in memory. Workflows with large histories consume more memory per cached workflow, reducing the number of workflows a single worker can handle efficiently.

## How?

[ContinueAsNew](terms/continue-as-new.md) works by completing the current workflow execution and immediately starting a new one with the same workflow ID, a fresh history, and whatever state you pass as the new input.

Here is a Go example of a long-running workflow that periodically continues as new:

```go
func SubscriptionWorkflow(ctx workflow.Context, state SubscriptionState) error {
    // Process signals, timers, activities as normal
    for {
        // ... do work ...

        // Check if it's time to continue as new
        if workflow.GetInfo(ctx).GetCurrentHistoryLength() > 10000 {
            return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
        }
    }
}
```

Consider triggering [ContinueAsNew](terms/continue-as-new.md) based on:
- **Event count**: When the history reaches a threshold (e.g. 10,000 events).
- **Elapsed time**: When the workflow has been running longer than a reasonable period (e.g. 24 hours). This caps the age of the code, simplifying versioning.
- **Explicit signal**: A signal sent specifically to trigger ContinueAsNew, useful for operational control.

When using ContinueAsNew, make sure to:
- **Carry over essential state**: Pass only the state the new execution needs as workflow input. This is your chance to compact state and drop anything no longer needed.
- **Drain pending work first**: If your workflow processes signals, drain the signal channel before continuing as new to avoid losing unprocessed signals. See [not draining signals before completing a workflow](<not-draining-signals-before-completing-workflow.md>).
- **Design the workflow input to serve as checkpoint**: Since the input to the new execution is whatever you pass to ContinueAsNew, the workflow input struct should be designed to capture resumable state from the start.
