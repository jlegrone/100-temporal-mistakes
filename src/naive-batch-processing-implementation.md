# Naive Batch Processing Implementation

> [!TIP]
> Concentrating all batch work in a single workflow causes [history overflow](overflowing-workflow-history-length.md) and [lock contention](workflow-lock-contention-due-to-concurrent-updates.md). Distribute work across [child workflows](terms/child-workflow.md) and pass references, not data.

Batch processing workflows that split data and process it all within a single workflow quickly hit Temporal's limits. Activities completing concurrently compete for the workflow lock, and large histories push against the 50k event limit.

Handle each batch in a separate child workflow. This spreads work across multiple histories and avoids lock contention. You can nest this -- batches of batches -- to create a tree of workflows with concurrency control at each level.

Keep data out of workflow inputs and outputs. All [payloads](terms/payload.md) are serialized, sent over the network, and stored in the backend. Pass batch IDs or references and let the leaf workflows look up data they need. Temporal is a control plane, not a data plane -- use it to orchestrate batch processing, but don't route the data itself through workflow payloads.
