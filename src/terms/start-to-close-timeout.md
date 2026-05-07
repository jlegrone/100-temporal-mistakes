# Start-to-Close Timeout

The start-to-close timeout limits the maximum execution time for a single activity attempt, measured from when the worker picks up the task until it reports a result. If the activity doesn't complete within this time, the attempt is considered failed and may be retried according to the retry policy, as long as the schedule-to-close timeout hasn't been reached.

Setting a start-to-close timeout (in addition to schedule-to-close) allows multiple retry attempts within the overall budget. Without it, there is no per-attempt timeout and a hung activity won't be retried until the schedule-to-close or heartbeat timeout is reached.

## Related

- [Preventing Activity Retries](../preventing-activity-retries.md)
- [Setting Too-Short Timeouts](../setting-too-short-timeouts.md)
- [Schedule-to-Close Timeout](schedule-to-close-timeout.md)
- [Heartbeat Timeout](heartbeat-timeout.md)
- [Retry Policy](retry-policy.md)
- [Activity Task](activity-task.md)
