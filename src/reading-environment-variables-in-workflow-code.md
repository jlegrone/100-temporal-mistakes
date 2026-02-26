# Reading Environment Variables in Workflow Code

> [!TIP]
> * Environment variables can differ between [workers](terms/worker.md) and between deployments, making workflow code [non-deterministic](terms/non-determinism.md) on [replay](terms/replay.md).
> * If you need configuration values in workflow logic, pass them as workflow input or use `SideEffect`/`MutableSideEffect`.
> * Environment variables are fine to read in activity code, worker initialization, or any code that runs outside the workflow function.

## What?

Reading environment variables directly in workflow code is a determinism violation. Workflow code is re-executed during [replay](terms/replay.md), and environment variables can return different values depending on which worker handles the replay and what the deployment environment looks like at that time.

```go
// BAD: reading env var in workflow code
func MyWorkflow(ctx workflow.Context) error {
    region := os.Getenv("AWS_REGION")
    if region == "us-east-1" {
        // Route to US activities
    } else {
        // Route to EU activities
    }
    // ...
}
```

If the workflow originally ran on a worker with `AWS_REGION=us-east-1` and later replays on a worker with `AWS_REGION=eu-west-1` (or after a deployment that changes the variable), the workflow takes a different code path and produces different commands, triggering a non-determinism error.

## Why?

Environment variables feel harmless because they look like simple constants. But they are external state that can change between:

- **Different workers.** In a heterogeneous deployment, workers may run in different regions, availability zones, or with different configurations.
- **Different deployments.** A deployment that updates an environment variable changes the value for all subsequent replays of existing workflows.
- **Different times.** Some environment variables are set dynamically (e.g., by container orchestrators) and may change across worker restarts.

The fundamental issue is the same as any other non-deterministic operation in workflow code: it produces different results on replay, breaking the [history](terms/event-history.md)-code contract that Temporal relies on.

## How?

**Pass configuration as workflow input.** This is the simplest and most explicit approach. The value is recorded in the workflow's start event and stays constant throughout the workflow's lifetime.

```go
type WorkflowInput struct {
    Region string
}

func MyWorkflow(ctx workflow.Context, input WorkflowInput) error {
    if input.Region == "us-east-1" {
        // Route to US activities -- deterministic
    } else {
        // Route to EU activities -- deterministic
    }
    // ...
}
```

**Use `SideEffect` for values that must be read at runtime.** `SideEffect` executes a function once, records the result in history, and returns the recorded value on replay.

```go
func MyWorkflow(ctx workflow.Context) error {
    var region string
    encoded := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
        return os.Getenv("AWS_REGION")
    })
    encoded.Get(&region)
    // region is now deterministic -- same value on replay
    // ...
}
```

**Use `MutableSideEffect` for values that may change and should be re-evaluated periodically** (rare in practice for environment variables, but available).

**Read environment variables in activities.** Activity code is not subject to determinism constraints and can freely read environment variables, make network calls, and perform any other non-deterministic operation.

**Read environment variables during worker initialization.** If you need configuration to register workflows or configure options, reading environment variables when the worker starts is perfectly safe -- this code is not replayed.
