# Workflow Lock Contention due to Concurrent Updates

> [!TIP]
> * Minimize concurrent updates to a given workflow history
> * Failing doing so lead to lock contention and high end-to-end workflow execution latency

## What?

Temporal scales well with the numbers of workflows but poorly when a given workflow receive many concurrent updates to its history.

When that happens you'll see contention in the form of a rise in `busy_workflow` errors from the `service_errors_resource_exhausted` metric and high end to end workflow execution latency.

## Why?

Workflow history updates are serialized using a workflow level locking mechanism. As a result, concurrent updates will eventually compete to acquire that workflow lock resulting in high end to end latency for your workflow executions due to the high overhead caused by code blocked waiting for lock acquisition.

Concurrent updates come in many flavors, the most obvious one being [signals](terms/signals.md) as those can be appended to workflow histories at any time by definition.
But asynchronous activities and child workflows completing at the same time, long running activities heartbeats, as well as workflows [updates](terms/updates.md) and [queries](terms/queries.md) can also lead to contention.

## How?

You should design your application to spread events over multiple workflows when possible.

You should avoid bad patterns like implementing a high throughput message queue over a single workflow or [naïve batch processing implementations](<naive-batch-processing-implementation.md>).
