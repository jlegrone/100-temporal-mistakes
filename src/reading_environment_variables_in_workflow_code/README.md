# Reading Environment Variables in Workflow Code

> [!TIP]
> Environment variables can differ between [workers](terms/worker.md) and between deployments, making them [non-deterministic](terms/non-determinism.md) on [replay](terms/replay.md). Pass configuration as workflow input or use `SideEffect` instead.

Reading environment variables directly in workflow code is a determinism violation. Environment variables can return different values depending on which worker handles the replay and what the deployment environment looks like at that time. They feel harmless because they look like constants, but they are external state that can change between different workers, deployments, and restarts.

<!--SNIPSTART reading-env-vars-bad-->
[reading_environment_variables_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/reading_environment_variables_in_workflow_code/workflow.go)
```go

// BAD: reading env var in workflow code
func MyWorkflowBad(ctx workflow.Context) error {
	region := os.Getenv("AWS_REGION")
	if region == "us-east-1" {
		// Route to US activities
	}
	// ...
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART reading-env-vars-good-->
[reading_environment_variables_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/reading_environment_variables_in_workflow_code/workflow.go)
```go

// GOOD: pass configuration as workflow input
type WorkflowInput struct {
	Region string
}

func MyWorkflowGood(ctx workflow.Context, input WorkflowInput) error {
	if input.Region == "us-east-1" {
		// Deterministic -- value is recorded in the start event
	}
	// ...
	return nil
}

```
<!--SNIPEND-->

For values that must be read at runtime, use `workflow.SideEffect` to capture and record them in [history](terms/event-history.md). Environment variables are fine to read in activity code, worker initialization, or any code that runs outside the workflow function.
