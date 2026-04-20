# Not Using ParentClosePolicy

> [!TIP]
> By default, [child workflows](terms/child-workflow.md) are [terminated](terms/terminate.md) (hard-killed) when their parent completes, fails, or is [canceled](terms/cancelation.md). Set `ParentClosePolicy` to `REQUEST_CANCEL` or `ABANDON` if children need cleanup.

The default `TERMINATE` policy silently kills child workflows with no opportunity to run cleanup logic, compensation activities, or defer blocks. This is a reasonable safety net against orphans, but the wrong choice when children need to release resources, send notifications, or complete in-flight work.

```go
childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
    ParentClosePolicy: enums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
})
future := workflow.ExecuteChildWorkflow(childCtx, ChildWorkflow, input)
```

| Policy | Behavior |
|---|---|
| `TERMINATE` (default) | Child is immediately killed. No cleanup. |
| `REQUEST_CANCEL` | Child receives a cancellation request and can handle it gracefully via a [disconnected context](not-using-disconnected-context-for-cleanup.md). |
| `ABANDON` | Child continues running independently, unaffected by parent closure. |

Choose `REQUEST_CANCEL` when the child should wind down gracefully. Choose `ABANDON` when the child's lifecycle is genuinely independent. Stick with `TERMINATE` only when you're certain the child has no cleanup needs.
