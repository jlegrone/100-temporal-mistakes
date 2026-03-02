# Naive Batch Processing Implementation

> [!TIP]
> * Temporal can process large data sets, but naive implementations hit history size limits and [workflow lock contention](workflow-lock-contention-due-to-concurrent-updates.md).
> * Minimize data flowing through workflow inputs and outputs -- pass references, not data.

## What?

Batch processing workflows take a list of data, split it into batches, and process each batch in sequence or in parallel. Without care, this quickly runs into Temporal's limits.

## Why?

Two problems arise in naive implementations. First, concentrating all work in a single workflow causes [workflow lock contention](workflow-lock-contention-due-to-concurrent-updates.md) as activities complete concurrently. Second, large histories push against the [history size limit](overflowing-workflow-history-size.md).

## How?

Handle each batch in a separate [child workflow](terms/child-workflow.md). This spreads work across multiple histories and avoids lock contention. You can nest this pattern -- batches of batches -- to create a tree of workflows with concurrency control at each level.

Keep data out of workflow inputs and outputs. All inputs and outputs are serialized, sent over the network, and stored in the Temporal backend. Instead, pass batch IDs or references and let the leaf workflows look up the data they need. This minimizes both bandwidth and storage.

## Alternatives

Temporal is a control plane, not a data plane. Use it to orchestrate batch processing, but don't route the data itself through workflow payloads.

For high-throughput, low-latency data pipelines, purpose-built systems like Kafka offer lower overhead -- at the cost of more complex orchestration and error handling.
