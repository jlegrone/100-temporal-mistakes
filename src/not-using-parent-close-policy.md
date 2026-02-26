# Not Using ParentClosePolicy

> [!TIP]
> * By default, child workflows are [terminated](terms/terminate.md) when their parent completes, fails, or is cancelled -- giving them no chance for cleanup.
> * If child workflows need to perform graceful cleanup, you must set `ParentClosePolicy` to `REQUEST_CANCEL` or `ABANDON`.
> * Choose the policy based on whether the child needs to react to the parent's closure or should simply continue independently.

## What?

When a parent workflow completes, fails, or is cancelled, Temporal applies a `ParentClosePolicy` to each of its running child workflows. The default policy is `TERMINATE`, which is the equivalent of a hard kill -- the child workflow stops immediately with no opportunity to run cleanup logic, defer blocks, or compensation activities.

Many developers start a child workflow expecting it to handle its own lifecycle gracefully, but never configure the `ParentClosePolicy`. When the parent finishes before the child, the child is silently terminated and any cleanup logic it would have run is lost.

## Why?

The default `TERMINATE` policy is a reasonable safety net: it prevents orphaned child workflows from running indefinitely after their parent is gone. But it is the wrong choice when child workflows need to:

- Release external resources (locks, reservations, temporary files).
- Send notifications or acknowledgements.
- Run compensation logic to undo partial work.
- Complete in-flight operations that must not be interrupted.

A terminated workflow has no chance to do any of this. Its pending activities are immediately abandoned and its code never executes another line. If your child workflow had important cleanup to perform, that cleanup simply does not happen.

## How?

Temporal offers three `ParentClosePolicy` values:

| Policy | Behavior |
|---|---|
| `TERMINATE` (default) | Child is immediately [terminated](terms/terminate.md). No cleanup. |
| `REQUEST_CANCEL` | Child receives a cancellation request and can handle it gracefully. |
| `ABANDON` | Child continues running independently, unaffected by the parent's closure. |

Set the policy when starting the child workflow:

```go
childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
    ParentClosePolicy: enums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
})
future := workflow.ExecuteChildWorkflow(childCtx, ChildWorkflow, input)
```

**Choose `REQUEST_CANCEL`** when the child should be notified that the parent is gone and should wind down gracefully. The child workflow will receive a cancellation signal and can use a [disconnected context](<not-using-disconnected-context-for-cleanup.md>) to perform cleanup activities before completing.

**Choose `ABANDON`** when the child workflow's lifecycle is genuinely independent and it should continue running regardless of what happens to the parent. Be mindful that abandoned child workflows can become long-lived orphans if they don't have their own termination conditions.

**Stick with `TERMINATE`** (or don't set anything) only when you are certain the child has no cleanup needs and you want the simplest possible behavior.
