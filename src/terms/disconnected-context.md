# Disconnected Context

A disconnected context is a workflow context that is not tied to the parent workflow's cancellation state. In the Go SDK, it is created using `workflow.NewDisconnectedContext(ctx)`. Other SDKs have equivalent mechanisms.

When a workflow is canceled, its main context is canceled, which means any new activities or child workflows started with that context will immediately fail. A disconnected context allows cleanup operations (compensation activities, notification activities, cleanup child workflows) to proceed even after the workflow has been canceled. It is essential to set a timeout on the disconnected context to prevent cleanup from running indefinitely.

## Related

- [Not Using Disconnected Context for Cleanup](../not_using_disconnected_context_for_cleanup/README.md)
- [Deadlocking When Workflow Canceled](../deadlocking_when_workflow_cancelled/)
- [Not Waiting for Child Workflows to Start](../not_waiting_for_child_workflows_to_start/README.md)
- [Cancellation](cancellation.md)
- [Child Workflow](child-workflow.md)
