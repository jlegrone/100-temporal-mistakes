# Unnecessary Child Workflows

> [!TIP]
> [Child workflows](terms/child-workflow.md) add a small amount of overhead compared to activities. Don't reach for them when a simple activity would suffice.

If you find yourself wrapping a single activity call in a child workflow, consider inlining the activity in the parent workflow instead.

Use a child workflow when you genuinely need independent lifecycle management (via [`ParentClosePolicy`](terms/parent-close-policy.md)), or logical isolation that will make it easier to reason about refactoring the workflow tasks (e.g. decomposing and activity into multiple activities) in the future.
