# Signals

Signals are asynchronous messages sent to a running workflow execution. They create events in the workflow history and can carry a payload. Unlike queries, signals can modify workflow state and trigger workflow logic. Signals are delivered at least once and are processed by the workflow in the order they appear in the history, though multiple signals sent concurrently may be recorded in any order.

## Related

- [Not draining signals before completing workflow](../not_draining_signals_before_completing_workflow/)
- [Assuming signal/update order](../assuming_signal_update_order/)
- [Workflow lock contention due to concurrent updates](../workflow-lock-contention-due-to-concurrent-updates.md)
- [Wrapping a queue with a workflow](../wrapping_a_queue_with_a_workflow/README.md)
- [Updates](updates.md)
- [Queries](queries.md)
- [Event History](event-history.md)
- [Payload](payload.md)
