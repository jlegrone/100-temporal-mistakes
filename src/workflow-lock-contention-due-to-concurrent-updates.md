# Workflow Lock Contention due to Concurrent Updates

> [!TIP]
> Temporal serializes [history](terms/event-history.md) updates through a per-workflow lock. Too many concurrent updates to a single workflow cause `busy_workflow` errors and high latency.

Temporal scales well across many workflows but poorly when a single workflow receives many concurrent updates. [Signals](terms/signals.md) are the most obvious source of contention since they arrive at any time, but activities and [child workflows](terms/child-workflow.md) completing simultaneously, [heartbeats](terms/heartbeat.md), [updates](terms/updates.md), and [queries](terms/queries.md) all compete for the same lock. Each blocked update adds latency to every workflow task waiting behind it.

Spread events across multiple workflows. Avoid funneling high-throughput work through a single workflow -- don't use one workflow as a [message queue](wrapping_a_queue_with_a_workflow/README.md) or build [naive batch processing](naive-batch-processing-implementation.md) that concentrates all work in one place.
