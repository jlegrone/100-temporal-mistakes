# Unnecessary Child Workflows

> [!TIP]
> * [Child workflows](terms/child-workflow.md) add coordination overhead, extra [history](terms/event-history.md) events, and more complex error handling compared to activities.
> * Don't reach for child workflows when a simple activity would suffice.
> * Use child workflows when you need independent lifecycle management, separate [retry policies](terms/retry-policy.md), history size management, or logical separation at the workflow level.

## What?

Child workflows are powerful, but teams sometimes use them as a default way to decompose work, even when a plain activity would do the job. A common pattern is wrapping a single activity call in a child workflow "just in case" or creating child workflows for every step because it "feels cleaner."

```go
// Unnecessary: a child workflow that just wraps an activity
func SendEmailChildWorkflow(ctx workflow.Context, email Email) error {
    return workflow.ExecuteActivity(ctx, SendEmail, email).Get(ctx, nil)
}

// In the parent workflow
func ParentWorkflow(ctx workflow.Context) error {
    // This child workflow adds overhead without meaningful benefit
    err := workflow.ExecuteChildWorkflow(ctx, SendEmailChildWorkflow, email).Get(ctx, nil)
    // ...
}
```

## Why?

Each child workflow comes with real costs:

1. **Additional history events**: Starting a child workflow generates `StartChildWorkflowExecutionInitiated`, `ChildWorkflowExecutionStarted`, and `ChildWorkflowExecutionCompleted` events in the parent's history, plus the child has its own full workflow history. An activity generates just `ActivityTaskScheduled` and `ActivityTaskCompleted`.
2. **Coordination overhead**: The parent and child workflows coordinate through the Temporal server. This adds latency and consumes server resources compared to scheduling an activity directly.
3. **More complex error handling**: Child workflow failures surface as `ChildWorkflowExecutionError` wrapping the underlying error. You now have two layers of retry configuration, two layers of timeout configuration, and two workflows to reason about when debugging failures.
4. **Harder to observe**: Instead of a straightforward workflow with activities, operators see a tree of workflows. Tracing a single logical operation requires jumping between multiple workflow histories.

These costs compound quickly when you have many unnecessary child workflows in a single parent.

## How?

**Use an activity** when the operation:
- Is a single unit of work (API call, database query, file operation)
- Doesn't need its own independent lifecycle
- Doesn't need different retry or timeout policies from the parent workflow
- Doesn't need to survive parent workflow [cancellation](terms/cancellation.md) independently

**Use a child workflow** when you genuinely need:
- **Independent lifecycle management**: The child should continue running even if the parent is cancelled or times out (using the [`ParentClosePolicy`](terms/parent-close-policy.md) option).
- **Separate retry policies**: The child needs fundamentally different retry behavior from the parent workflow's activities.
- **History size management**: The parent workflow is processing large batches and would overflow its history limit without distributing work across child workflows. See [overflowing workflow history size](overflowing-workflow-history-size.md).
- **Logical domain boundary**: The child represents a genuinely independent business process (e.g., an "order fulfillment" sub-process within a larger "purchase" workflow) that might be triggered independently in other contexts.
- **Different [task queue](terms/task-queue.md)**: The child needs to run on a different set of [workers](terms/worker.md) than the parent.

When in doubt, start with an activity. You can always refactor to a child workflow later if the need arises. Going the other way -- collapsing unnecessary child workflows back into activities -- is harder because callers may already depend on the child workflow's independent existence.
