# Disconnected Context

A disconnected context is a workflow context that is not tied to the parent workflow's cancellation state. In the Go SDK, it is created using `workflow.NewDisconnectedContext(ctx)`. Other SDKs have equivalent mechanisms.

When a workflow is cancelled, its main context is cancelled, which means any new activities or child workflows started with that context will immediately fail. A disconnected context allows cleanup operations (compensation activities, notification activities, cleanup child workflows) to proceed even after the workflow has been cancelled. It is essential to set a timeout on the disconnected context to prevent cleanup from running indefinitely.

## Related

- [Not Using Disconnected Context for Cleanup](../not-using-disconnected-context-for-cleanup.md)
- [Deadlocking When Workflow Cancelled](../deadlocking-when-workflow-cancelled.md)
- [Not Waiting for Child Workflows to Start](../not-waiting-for-child-workflows-to-start.md)
- [Cancellation](cancellation.md)
- [Child Workflow](child-workflow.md)
