# Schedule-to-Close Timeout

The schedule-to-close timeout limits the total time from when an activity is scheduled until it must complete successfully, including all retries. If this timeout is reached, the activity is considered failed regardless of how many retry attempts remain. It represents the total budget for the activity to produce a result.

This timeout is important for preventing infinite retry loops. Setting it to the same value as the start-to-close timeout effectively limits the activity to a single attempt, because there's no time remaining for retries after the first attempt fails.

## Related

- [Preventing Activity Retries](../preventing-activity-retries.md)
- [Setting Too-Short Timeouts](../setting-too-short-timeouts.md)
- [Start-to-Close Timeout](start-to-close-timeout.md)
- [Heartbeat Timeout](heartbeat-timeout.md)
- [Retry Policy](retry-policy.md)
- [Activity Task](activity-task.md)
