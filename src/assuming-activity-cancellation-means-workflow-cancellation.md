# Assuming Activity Cancellation Means Workflow Cancellation

> [!TIP]
> * Activities can be cancelled for several reasons: heartbeat timeout, start-to-close timeout, worker shutdown, explicit cancellation from the workflow, or workflow cancellation.
> * Handling activity cancellation by assuming the whole workflow is cancelled can lead to premature workflow termination or skipped work.
> * Always check the actual cancellation reason before deciding what to do.

## What?

When an activity receives a cancellation signal via its context, it is tempting to assume that the parent workflow has been cancelled. After all, workflow cancellation is the most dramatic reason for an activity to be interrupted. But activities can be cancelled for many reasons that have nothing to do with the workflow's lifecycle:

- **Heartbeat timeout**: the activity failed to heartbeat within its configured `HeartbeatTimeout` and the server considered it stale.
- **Start-to-close timeout**: the activity exceeded its `StartToCloseTimeout`.
- **Worker shutdown**: the worker is shutting down gracefully and cancelled all in-flight activities to give them a chance to clean up before the process exits.
- **Explicit cancellation from the workflow**: the workflow logic decided to cancel a specific activity (e.g. a race pattern where the first result wins and the others are cancelled).

In none of those cases is the workflow itself cancelled.

## Why?

If your activity cancellation handler assumes the workflow is done and acts accordingly -- for example, by skipping compensation logic, by not returning partial results, or by treating the situation as a terminal error -- you'll introduce subtle bugs. The workflow may still be running and waiting for the activity to report back. Returning a misleading error or swallowing a result can cause the workflow to take an incorrect code path, retry unnecessarily, or hang.

This mistake is particularly insidious because it often works fine in tests (where cancellation usually does come from the workflow) and only surfaces in production under timeout or deployment scenarios.

## How?

In Go, the `temporal.IsCanceledError` helper tells you that the context was cancelled but not _why_. To distinguish the reason, you need to check the activity's context and the error type returned by heartbeating:

1. **Check the context error first.** If `ctx.Err()` returns `context.Canceled`, cancellation was requested. If it returns `context.DeadlineExceeded`, a timeout fired.
2. **Don't conflate activity cancellation with workflow cancellation.** In your activity code, treat cancellation as "stop what you're doing and return" rather than "the world is ending." Return partial results or a well-defined error that the workflow can interpret.
3. **Let the workflow decide what cancellation means.** The workflow has the full picture. It knows whether it was cancelled, whether it timed out the activity, or whether it explicitly requested cancellation. Design your activity to report what happened and let the workflow orchestrate the response.

```go
func MyActivity(ctx context.Context, input Input) (Result, error) {
    for {
        // Do work in increments...

        // Heartbeat regularly
        activity.RecordHeartbeat(ctx, progressInfo)

        // Check if we've been asked to stop
        if ctx.Err() != nil {
            // Return partial results -- let the workflow decide what to do
            return Result{Partial: true, Progress: progressInfo}, ctx.Err()
        }
    }
}
```

The key insight is that the activity should not try to be smarter than the workflow. Report what happened, return what you have, and let the orchestration layer handle the rest.
