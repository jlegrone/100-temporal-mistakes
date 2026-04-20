# Cancellation

Cancellation in Temporal is a cooperative mechanism for requesting that a workflow or activity stop its work. Unlike termination, cancellation gives the target a chance to perform cleanup operations (compensation, resource release, notifications) before completing.

When a workflow is cancelled, the SDK cancels the workflow's context, which propagates to all pending activities and child workflows. The workflow can catch this cancellation and run cleanup logic using a disconnected context. Activity cancellation is delivered via the heartbeat mechanism -- the server informs the activity of cancellation in the response to its next heartbeat.

## Related

- [Terminating Rather Than Canceling](../terminating-rather-than-canceling.md)
- [Deadlocking When Workflow Cancelled](../deadlocking-when-workflow-cancelled.md)
- [Not Using Disconnected Context for Cleanup](../not-using-disconnected-context-for-cleanup.md)
- [Assuming Activity Cancellation Means Workflow Cancellation](../assuming-activity-cancelation-means-workflow-cancelation.md)
- [Not Sending Heartbeats for Cancellation](../not-sending-heartbeats-for-cancellation.md)
- [Terminate](terminate.md)
- [Heartbeat](heartbeat.md)
- [Disconnected Context](disconnected-context.md)
