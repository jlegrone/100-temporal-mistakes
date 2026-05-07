# Heartbeat Timeout

The heartbeat timeout is the maximum time the Temporal server will wait between heartbeats from a running activity before considering it failed. If a heartbeat is not received within this window, the server marks the current attempt as timed out and schedules a retry (if retries remain).

The heartbeat timeout is distinct from the start-to-close timeout. It provides faster failure detection for long-running activities: instead of waiting for the full start-to-close timeout, a stuck or crashed activity is detected within one heartbeat interval. It also enables activity cancellation, as cancellation signals are delivered via heartbeat responses.

## Related

- [Preventing Activity Retries](../preventing-activity-retries.md)
- [Not Sending Heartbeats from Activities You Want to Cancel](../not_sending_heartbeats_for_cancellation/README.md)
- [Not Using Activity Heartbeat Details](../not_using_activity_heartbeat_details/README.md)
- [Heartbeat](heartbeat.md)
- [Start-to-Close Timeout](start-to-close-timeout.md)
- [Schedule-to-Close Timeout](schedule-to-close-timeout.md)
- [Activity Task](activity-task.md)
- [Cancellation](cancellation.md)
