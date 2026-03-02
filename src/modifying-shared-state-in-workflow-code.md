# Modifying Shared State in Workflow Code

> [!TIP]
> * Workflow code runs in a shared [worker](terms/worker.md) process -- modifying global variables, singletons, or shared maps creates race conditions and breaks determinism.
> * Multiple workflow executions run concurrently on the same worker, so shared mutable state is accessed from multiple goroutines/threads without synchronization.
> * Keep all workflow state local to the workflow function. Use activities for anything that needs to interact with external or shared state.

## What?

Modifying shared state -- global variables, package-level variables, singletons, static fields, shared maps or caches -- from within workflow code is a mistake that causes two distinct problems: race conditions between concurrent workflow executions and [non-determinism](terms/non-determinism.md) on [replay](terms/replay.md).

A Temporal worker runs many workflow executions concurrently in the same process. When workflow code writes to a global variable, every concurrent workflow execution on that worker is racing to read and write the same memory without synchronization.

```go
// BAD: shared mutable state
var processedCount int

func MyWorkflow(ctx workflow.Context) error {
    // Multiple workflow executions increment this concurrently -- data race!
    processedCount++
    if processedCount > 100 {
        // This condition depends on how many other workflows ran -- non-deterministic!
        workflow.ExecuteActivity(ctx, AlertActivity).Get(ctx, nil)
    }
    // ...
}
```

## Why?

**Race conditions.** In Go, concurrent access to a shared variable without synchronization is a data race, which is undefined behavior. In other languages, similar issues arise. Even if the program doesn't crash, the values read and written are unpredictable.

**Non-determinism on replay.** Global state depends on which workflows have executed and in what order. When a workflow replays, the global state is in a completely different condition than it was during the original execution. The workflow reads a different value, makes a different decision, and produces a non-determinism error.

**Invisible coupling.** Shared state creates hidden dependencies between workflow executions. One workflow's behavior changes based on what other workflows have done. This makes reasoning about individual workflow behavior extremely difficult and makes bugs nearly impossible to reproduce.

**Testing becomes unreliable.** Tests that run workflows sequentially may pass, while production (where workflows run concurrently) fails. The test environment can't reproduce the exact interleaving of concurrent executions.

## How?

**Keep state local to the workflow function.** All variables that a workflow needs should be declared within the workflow function or passed as parameters. This guarantees each workflow execution has its own isolated state.

```go
// GOOD: local state
func MyWorkflow(ctx workflow.Context) error {
    processedCount := 0 // Local to this workflow execution
    // ...
}
```

**Use workflow state for data that persists across events.** Variables declared in the workflow function naturally survive replay because the function re-executes and the values are reconstructed from [history](terms/event-history.md). No external storage is needed to maintain workflow state.

**Use activities for external state.** If your workflow needs to read from or write to a shared resource (a database, a cache, a counter service), do it through an activity. The activity result is recorded in history and replayed deterministically.

**Pass dependencies through workflow input or activity results.** Instead of reading configuration from a global singleton, pass it as workflow input. Instead of writing to a shared metrics map, emit metrics from an activity or from worker-level code outside the workflow function.

**Be cautious with injected dependencies.** Even if you avoid global variables, injecting a shared service (like a logger with mutable state or a shared cache) into workflow code can cause the same problems if the workflow modifies that shared object. Ensure injected dependencies are either read-only or scoped per workflow execution.
