# Starting Workflows from Activities

> [!TIP]
> * Activities are meant for side effects like calling external services, not for Temporal API operations like starting workflows.
> * Starting a workflow from an activity hides the workflow creation from the parent workflow's history and risks duplicate workflows on retry.
> * Start [child workflows](terms/child-workflow.md) directly from workflow code, or use the SDK client from outside a [worker](terms/worker.md).

## What?

A common mistake is using an activity to start another workflow by calling the Temporal SDK client from within the activity function. While this technically works, it misuses the purpose of activities and introduces subtle problems.

Activities are designed for interactions with the outside world -- calling APIs, reading files, writing to databases. Starting a Temporal workflow is an internal platform operation that has first-class support in workflow code via child workflows.

## Why?

When you start a workflow from an activity, several things go wrong:

1. **Invisible to the parent workflow.** The child workflow start doesn't appear in the parent workflow's [event history](terms/event-history.md). There's no parent-child relationship tracked by Temporal, so you lose visibility into the relationship between the two workflows.

2. **Duplicate workflows on retry.** If the activity fails after starting the workflow (e.g., network timeout on the response), the activity will be retried and attempt to start the workflow again. Unless you've carefully set a deterministic [workflow ID](terms/workflow-id.md) with a dedup policy, you'll end up with duplicate workflows.

3. **No [cancellation](terms/cancellation.md) propagation.** Parent-child workflow cancellation propagation is a built-in feature of Temporal, but only works with proper child workflows. A workflow started from an activity is completely detached from the parent.

4. **No result forwarding.** With child workflows, the parent can await the child's result directly. With an activity-started workflow, you'd need to build your own mechanism to get the result back.

## How?

**From workflow code**, use child workflows:

```go
// Good: start a child workflow directly from workflow code
childFuture := workflow.ExecuteChildWorkflow(ctx, MyChildWorkflow, input)
var result MyResult
err := childFuture.Get(ctx, &result)
```

```go
// Bad: start a workflow from an activity
func MyActivity(ctx context.Context, input MyInput) error {
    c, _ := client.Dial(client.Options{})
    _, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, SomeWorkflow, input)
    return err
}
```

**From outside a worker** (e.g., an HTTP handler, a cron job), using the SDK client to start workflows is perfectly fine -- that's what it's for. The anti-pattern is specifically about using the client inside an activity when a child workflow would be more appropriate.

If you genuinely need to start a workflow from an activity (rare cases where you explicitly don't want a parent-child relationship), make sure to use a deterministic workflow ID to handle retries safely.
