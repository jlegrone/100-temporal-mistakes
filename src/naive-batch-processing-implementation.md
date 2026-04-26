# Naive Batch Processing Implementation

> [!TIP]
> Concentrating all batch work in a single workflow causes [history overflow](overflowing-workflow-history-length.md) and [lock contention](workflow-lock-contention-due-to-concurrent-updates.md). Distribute work across [child workflows](terms/child-workflow.md) instead.

Batch processing workflows that process data all within a single workflow execution may quickly hit Temporal's limits. Activities scheduled concurrently compete for the workflow lock, and large histories push against the 50k event limit.
<!-- TODO: Link to both workflow history size and length limits, and lock contention mistake entries -->

Split work into size limited batches and handle each batch in a separate child workflow. This spreads work across multiple workflow executions and avoids lock contention. You can nest this -- batches of batches -- to create a tree of workflows with concurrency control at each level.

<!-- TODO: Also link to large individual payload limit -- be careful not to aggregate results from batches into one large payload. -->
