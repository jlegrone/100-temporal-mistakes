# Modifying Shared State in Workflow Code

<!-- Workflow code must not access or modify variables outside of its direct control flow. In go, an example would be reading the value of a package level variable (which could change between deployments). Or defining a workflow as a method on a struct, and writing/reading fields on that struct in the workflow code. In cases where you need shared state, use an external database or cache and read/write values via activities. -->

> [!TIP]
> Workflow code runs in a shared [worker](terms/worker.md) process -- modifying global variables, singletons, or shared maps creates race conditions and breaks determinism on [replay](terms/replay.md).

A Temporal worker runs many workflow executions concurrently in the same process. When workflow code writes to a global variable, every concurrent execution races to read and write the same memory without synchronization. On replay, the global state is in a completely different condition than it was during the original execution, causing [non-determinism](terms/non-determinism.md) errors.

<!-- TODO: refactor these examples to define the workflows as struct methods and a struct field instead of using package level variable -->
<!--SNIPSTART modifying-shared-state-bad-->
[modifying_shared_state_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/modifying_shared_state_in_workflow_code/workflow.go)
```go

// BAD: shared mutable state
var processedCount int

func MyWorkflowV1(ctx workflow.Context) error {
	processedCount++ // Data race! Non-deterministic on replay!
	if processedCount > 100 {
		if err := workflow.ExecuteActivity(ctx, AlertActivity).Get(ctx, nil); err != nil {
			return err
		}
	}
	// ...
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART modifying-shared-state-good-->
[modifying_shared_state_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/modifying_shared_state_in_workflow_code/workflow.go)
```go

// GOOD: local state
func MyWorkflowV2(ctx workflow.Context) error {
	processedCount := 0 // Local to this workflow execution
	_ = processedCount
	// ...
	return nil
}

```
<!--SNIPEND-->

Keep all workflow state local to the workflow function. Use activities for anything that needs to interact with external or shared state (databases, caches, counters). Be cautious with injected dependencies -- a shared logger with mutable state or a shared cache can cause the same problems if the workflow modifies that shared object.
