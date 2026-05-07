# Reading Environment Variables in Workflow Code

<!-- TODO: inline the LookupEnv workflow helper via snip and use that in the v2 example. Note that passing configuration via workflow inputs is another good alternative, but show the safe way to read env vars via side effect first. -->

> [!TIP]
> Environment variables values can differ between worker versions, making them [non-deterministic](terms/non-determinism.md) on [replay](terms/replay.md). Pass configuration as workflow input or use `SideEffect` to read environment variables instead.

Reading environment variables directly in workflow code is a determinism violation. Environment variables can return different values depending on which worker handles the replay and what the deployment environment looks like at that time. They feel harmless because they look like constants, but they are external state that can change between different worker instances and deployment versions.

<!--SNIPSTART reading-env-vars-bad-->
[reading_environment_variables_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/reading_environment_variables_in_workflow_code/workflow.go)
```go

// BAD: reading env var in workflow code
func MyWorkflowV1(ctx workflow.Context) error {
	region := os.Getenv("AWS_REGION")
	if region == "us-east-1" {
		// Route to US activities
	}
	// ...
	return nil
}

```
<!--SNIPEND-->

<!-- TODO: Delete the workflow input example and add a side effect example (using LookupEnv helper which should also be included in a separate snippet) -->

<!--SNIPSTART reading-env-vars-good-->
[reading_environment_variables_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/reading_environment_variables_in_workflow_code/workflow.go)
```go

// GOOD: pass configuration as workflow input
type WorkflowInput struct {
	Region string
}

func MyWorkflowV2(ctx workflow.Context, input WorkflowInput) error {
	if input.Region == "us-east-1" {
		// Deterministic -- value is recorded in the start event
	}
	// ...
	return nil
}

```
<!--SNIPEND-->
