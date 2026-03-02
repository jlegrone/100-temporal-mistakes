# Workflow Lock Contention due to Concurrent Updates

> [!TIP]
> * Minimize concurrent updates to any single workflow.
> * Too many concurrent updates cause lock contention and high end-to-end latency.

## What?

Temporal scales well across many workflows but poorly when a single workflow receives many concurrent updates to its [history](terms/event-history.md).

When this happens, you'll see `busy_workflow` errors from the `service_errors_resource_exhausted` metric and high end-to-end workflow execution latency.

## Why?

Temporal serializes history updates through a per-workflow lock. Concurrent updates compete for this lock, and each blocked update adds latency to every workflow task waiting behind it.

[Signals](terms/signals.md) are the most obvious source of contention since they can arrive at any time. But activities and [child workflows](terms/child-workflow.md) completing simultaneously, long-running activity [heartbeats](terms/heartbeat.md), [updates](terms/updates.md), and [queries](terms/queries.md) all compete for the same lock.

## How?

Spread events across multiple workflows. Avoid funneling high-throughput work through a single workflow -- don't use one workflow as a message queue or build [naive batch processing implementations](naive-batch-processing-implementation.md) that concentrate all work in one place.
