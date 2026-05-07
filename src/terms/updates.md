# Updates

Updates are a synchronous communication mechanism for workflows, combining the write capability of signals with a return value. An update sends data to a running workflow, the workflow processes it, and the caller receives a response. Updates can include a validation step that runs before the update is accepted into the history, allowing the workflow to reject invalid updates. Updates create events in the workflow history.

## Related

- [Assuming signal/update order](../assuming_signal_update_order/)
- [Workflow lock contention due to concurrent updates](../workflow-lock-contention-due-to-concurrent-updates.md)
- [Signals](signals.md)
- [Queries](queries.md)
- [Event History](event-history.md)
