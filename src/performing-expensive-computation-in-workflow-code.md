# Performing Expensive Computation in Workflow Code

> [!TIP]
> * Workflow tasks have a default 10-second execution timeout. If your workflow code takes too long to execute, the task times out and gets retried, potentially causing a livelock.
> * Expensive computation is re-executed on every [replay](terms/replay.md), compounding the performance problem.
> * Move CPU-intensive or long-running computation into activities, which have independent timeouts and don't block the workflow task.

## What?

Performing expensive or long-running computation directly in workflow code -- large data transformations, complex mathematical calculations, heavy string processing, parsing large files from memory -- can cause the workflow task to exceed its execution timeout. By default, a workflow task must complete within 10 seconds. If it doesn't, the Temporal server considers the task lost and schedules it again on (potentially) another worker.

The retried task starts from the beginning (replaying the full history), hits the same expensive computation, times out again, and the cycle repeats. This creates a livelock where the workflow is perpetually retrying but never making progress. For more on this failure mode, see [Exceeding the 10s Workflow Task Timeout](exceeding-10s-task-timeout.md).

```go
// BAD: expensive computation in workflow code
func MyWorkflow(ctx workflow.Context, data []Record) error {
    // This takes 30 seconds for large datasets
    result := expensiveTransformation(data)
    err := workflow.ExecuteActivity(ctx, StoreResult, result).Get(ctx, nil)
    return err
}
```

## Why?

**Livelock.** The most severe consequence. The workflow task times out, gets retried, times out again, and the workflow never makes progress. Unlike a stuck activity that can be independently timed out and retried, a stuck workflow task blocks all forward progress for the entire workflow execution.

**Replay amplification.** Expensive computation in workflow code runs on every replay. If a workflow replays 10 times over its lifetime (due to worker restarts, deployments, cache evictions), the expensive computation runs 10 times. If it takes 5 seconds each time, that's 50 seconds of wasted CPU -- and each replay gets slower as history grows because more events must be processed before reaching the computation.

**Worker resource contention.** Workflow tasks run on the workflow task poller, which typically has limited concurrency. A long-running computation occupies one of these slots, reducing the worker's ability to process other workflow tasks. This can cascade and affect all workflows on that worker.

**The 10-second timeout is intentionally short.** Workflow tasks are meant to be lightweight orchestration logic: make a decision, schedule some activities or timers, and yield. The timeout reflects this design intent. Fighting the timeout by increasing it is almost always the wrong solution.

## How?

**Move computation into activities.**

```go
// GOOD: expensive computation in an activity
func TransformActivity(ctx context.Context, data []Record) (Result, error) {
    // Activities have their own configurable timeouts
    return expensiveTransformation(data), nil
}

func MyWorkflow(ctx workflow.Context, data []Record) error {
    var result Result
    // Activity has a generous timeout, can heartbeat, and won't block the workflow task
    actOpts := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Minute,
        HeartbeatTimeout:    30 * time.Second,
    }
    ctx = workflow.WithActivityOptions(ctx, actOpts)
    err := workflow.ExecuteActivity(ctx, TransformActivity, data).Get(ctx, &result)
    if err != nil {
        return err
    }
    err = workflow.ExecuteActivity(ctx, StoreResult, result).Get(ctx, nil)
    return err
}
```

Activities have their own independently configurable timeouts, can heartbeat to report progress, and their results are recorded in history so the computation doesn't repeat on [replay](terms/replay.md).

**Use local activities for lighter computation.** If the computation is not too heavy (a few hundred milliseconds) but still not suitable for workflow code, local activities are a lighter-weight option. They execute in the same worker process, avoid the overhead of scheduling through the server, but still record their result in history.

**If computation is truly trivial, keep it in workflow code.** Simple comparisons, basic arithmetic, string formatting, or building an activity input from workflow state -- these are fine in workflow code. The rule of thumb: if it completes in well under a second, it can stay in the workflow function. If it might take seconds, move it to an activity.
