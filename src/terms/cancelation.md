# Cancelation

Cancelation in Temporal is a cooperative mechanism for requesting that a workflow or activity stop its work. Unlike termination, cancelation gives the target a chance to perform cleanup operations (compensation, resource release, notifications) before completing.

When a workflow is canceled, the SDK cancels the workflow's context, which propagates to all pending activities and child workflows. The workflow can catch this cancelation and run cleanup logic using a disconnected context. Activity cancelation is delivered via the heartbeat mechanism -- the server informs the activity of cancelation in the response to its next heartbeat.

## Related

- [Terminating Rather Than Canceling](../terminating-rather-than-canceling.md)
- [Deadlocking When Workflow Canceled](../deadlocking-when-workflow-canceled.md)
- [Not Using Disconnected Context for Cleanup](../not-using-disconnected-context-for-cleanup.md)
- [Assuming Activity Cancelation Means Workflow Cancelation](../assuming-activity-cancelation-means-workflow-cancelation.md)
- [Not Sending Heartbeats for Cancelation](../not-sending-heartbeats-for-cancelation.md)
- [Terminate](terminate.md)
- [Heartbeat](heartbeat.md)
- [Disconnected Context](disconnected-context.md)
