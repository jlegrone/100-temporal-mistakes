# Retry Policy

A retry policy defines how Temporal retries failed activities or workflows. It includes parameters like initial interval, backoff coefficient, maximum interval, maximum attempts, and non-retryable error types. Retry policies are configured when scheduling an activity or starting a workflow.

For activities, retries are managed by the server -- when an activity fails, the server schedules a new attempt according to the retry policy. For workflows, retries restart the entire execution from scratch, discarding all accumulated state, which is why workflow retries are rarely appropriate.

## Related

- [Preventing Activity Retries](../preventing-activity-retries.md)
- [Using Workflow Retries](../using-workflow-retries.md)
- [Setting Too Short Timeouts](../setting-too-short-timeouts.md)
- [Not Making Activities Idempotent](../not-making-activities-idempotent.md)
- [Activity Task](activity-task.md)
- [Schedule-to-Close Timeout](schedule-to-close-timeout.md)
- [Start-to-Close Timeout](start-to-close-timeout.md)
