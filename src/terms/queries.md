# Queries

Queries are read-only operations that retrieve the current state of a workflow execution. Unlike signals and updates, queries do not modify workflow state or create events in the history. A query handler runs within the workflow code context and has access to the workflow's local variables. For closed workflows, the entire history must be replayed to reconstruct the state before the query handler can execute.

## Related

- [Querying closed workflows](../querying-closed-workflows.md)
- [Workflow lock contention due to concurrent updates](../workflow-lock-contention-due-to-concurrent-updates.md)
- [Signals](signals.md)
- [Updates](updates.md)
- [Replay](replay.md)
- [Event History](event-history.md)
