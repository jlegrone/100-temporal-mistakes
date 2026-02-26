# Heartbeat

Heartbeating is the mechanism by which long-running activities report progress back to the Temporal server. An activity sends heartbeats at regular intervals to indicate it is still alive and making progress. If the server doesn't receive a heartbeat within the configured heartbeat timeout, it considers the activity failed and schedules a retry.

Heartbeats serve three purposes: liveness detection (the server knows the activity is still running), cancellation delivery (the server communicates cancellation requests via heartbeat responses), and progress checkpointing (heartbeat details can carry progress state that a retried activity can resume from).

## Related

- [Not Sending Heartbeats for Cancellation](../not-sending-heartbeats-for-cancellation.md)
- [Not Using Activity Heartbeat Details](../not-using-activity-heartbeat-details.md)
- [Preventing Activity Retries](../preventing-activity-retries.md)
- [Not Draining Activity Tasks Before Shutdown](../not-draining-activity-tasks-before-shutdown.md)
- [Activity Task](activity-task.md)
- [Heartbeat Timeout](heartbeat-timeout.md)
- [Start-to-Close Timeout](start-to-close-timeout.md)
