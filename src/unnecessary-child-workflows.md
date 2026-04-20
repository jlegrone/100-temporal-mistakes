# Unnecessary Child Workflows

> [!TIP]
> [Child workflows](terms/child-workflow.md) add coordination overhead, extra [history](terms/event-history.md) events, and more complex error handling compared to activities. Don't reach for them when a simple activity would suffice.

A common pattern is wrapping a single activity call in a child workflow "just in case." Each child workflow generates extra events in the parent's history, adds a round-trip to the server for coordination, introduces two layers of retry/timeout configuration, and makes tracing require jumping between multiple workflow histories.

Use an activity when the operation is a single unit of work that doesn't need its own independent lifecycle. Use a child workflow when you genuinely need: independent lifecycle management (via [`ParentClosePolicy`](terms/parent-close-policy.md)), history size management for large batches, a logical domain boundary that might be triggered independently, or execution on a different [task queue](terms/task-queue.md).

When in doubt, start with an activity. Refactoring to a child workflow later is easier than collapsing unnecessary child workflows back into activities.
