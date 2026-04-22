# Not Using ParentClosePolicy

<!-- TODO: Consider suggesting a linter setting that makes parent close policy required, or an interceptor that changes the default to REQUEST_CANCEL. Cross-link to the cancelation deadlock mistake, which can cause dangling workflow executions after cancelation is requested. If the interceptor approach seems too heavy handed, it could also enforce that SOME policy is set (you would just discover this at runtime (ideally in tests). Add an example go interceptor and a unit test that confirms that a child workflow is rejected with an error by the interceptor if policy is not specified. The interceptor should have a toggleable "enforce only" and "apply default" mode. -->
<!-- TODO: Provide additional guidance around the usage of the "abandon" policy -- it may be useful if the child workflow is meant to run to perform some state reconciliation that must complete without interference from the parent workflow. But this is a special case and abandon workflows should be thoroughly tested to ensure they eventually run to completion (or, set a timeout). -->

> [!TIP]
> By default, [child workflows](terms/child-workflow.md) are [terminated](terms/terminate.md) (hard-killed) when their parent completes, fails, or is [canceled](terms/cancelation.md). Set `ParentClosePolicy` to `REQUEST_CANCEL` or `ABANDON` if children need cleanup.

The default `TERMINATE` policy silently kills child workflows with no opportunity to run cleanup logic, compensation activities, or defer blocks. This is a reasonable safety net against orphans, but the wrong choice when children need to release resources, send notifications, or complete in-flight work.

<!--SNIPSTART not-using-parent-close-policy-workflow-->
[not_using_parent_close_policy/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_parent_close_policy/workflow.go)
```go

func MyWorkflow(ctx workflow.Context, input Input) error {
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_REQUEST_CANCEL,
	})
	future := workflow.ExecuteChildWorkflow(childCtx, ChildWorkflow, input)
	return future.Get(childCtx, nil)
}

```
<!--SNIPEND-->

| Policy | Behavior |
|---|---|
| `TERMINATE` (default) | Child is immediately killed. No cleanup. |
| `REQUEST_CANCEL` | Child receives a cancellation request and can handle it gracefully via a [disconnected context](../not_using_disconnected_context_for_cleanup/README.md). |
| `ABANDON` | Child continues running independently, unaffected by parent closure. |

Choose `REQUEST_CANCEL` when the child should wind down gracefully. Choose `ABANDON` when the child's lifecycle is genuinely independent. Stick with `TERMINATE` only when you're certain the child has no cleanup needs.
