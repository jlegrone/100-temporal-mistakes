# Reading Environment Variables in Workflow Code

> [!TIP]
> Environment variables can differ between [workers](terms/worker.md) and between deployments, making them [non-deterministic](terms/non-determinism.md) on [replay](terms/replay.md). Pass configuration as workflow input or use `SideEffect` instead.

Reading environment variables directly in workflow code is a determinism violation. Environment variables can return different values depending on which worker handles the replay and what the deployment environment looks like at that time. They feel harmless because they look like constants, but they are external state that can change between different workers, deployments, and restarts.

```go
// BAD: reading env var in workflow code
func MyWorkflow(ctx workflow.Context) error {
    region := os.Getenv("AWS_REGION")
    if region == "us-east-1" {
        // Route to US activities
    }
    // ...
}

// GOOD: pass configuration as workflow input
type WorkflowInput struct {
    Region string
}

func MyWorkflow(ctx workflow.Context, input WorkflowInput) error {
    if input.Region == "us-east-1" {
        // Deterministic -- value is recorded in the start event
    }
    // ...
}
```

For values that must be read at runtime, use `workflow.SideEffect` to capture and record them in [history](terms/event-history.md). Environment variables are fine to read in activity code, worker initialization, or any code that runs outside the workflow function.
