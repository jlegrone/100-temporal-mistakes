# Workflow Lock Contention due to Concurrent Updates

> [!TIP]
> Too many concurrent updates to a single workflow can cause `busy_workflow` errors and high latency because of the way Temporal serializes [history](terms/event-history.md) updates through a server side lock.

Temporal scales well across many workflows but poorly when a single workflow receives many concurrent updates. [Signals](terms/signals.md) can be a difficult source of contention since they are often sent in response to external events. Other causes for contention can be activities or [child workflows](terms/child-workflow.md) completing simultaneously, [updates](terms/updates.md), and [queries](terms/queries.md).

Avoid funneling high-throughput tasks through a single workflow by spreading events across multiple workflows when possible.

<!-- TODO: Add instructions for how to monitor for lock contention (maybe not possible in temporal cloud?) -->
<!-- TODO: Link to batch entry, fan-out workflow design pattern. -->
