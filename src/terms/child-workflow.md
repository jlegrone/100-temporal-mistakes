# Child Workflow

A child workflow is a workflow execution started from within another workflow (the parent). Child workflows have their own event history, independent retry policies, and can be on different task queues. The parent-child relationship is recorded in both histories, and the parent can wait for the child's result.

Child workflows are useful for breaking down large workflows into smaller pieces (avoiding history size limits), providing independent failure isolation, enabling different versioning lifecycles, and working around the constraint that a single workflow's history has size limits.

The `ParentClosePolicy` controls what happens to child workflows when the parent completes or is canceled/terminated: `TERMINATE` (default) immediately terminates the child, `REQUEST_CANCEL` sends a cancellation request, and `ABANDON` lets the child continue running independently.

## Related

- [Unnecessary Child Workflows](../unnecessary-child-workflows.md)
- [Not Using Parent Close Policy](../not_using_parent_close_policy/README.md)
- [Not Waiting for Child Workflows to Start](../not_waiting_for_child_workflows_to_start/README.md)
- [Doing Too Many Things in One Workflow](../doing-too-many-things-in-one-workflow.md)
- [Polling Workflow Results](../polling_workflow_results/)
- [Parent Close Policy](parent-close-policy.md)
- [Event History](event-history.md)
- [Task Queue](task-queue.md)
