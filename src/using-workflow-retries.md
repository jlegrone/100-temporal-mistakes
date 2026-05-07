# Using Workflow Retries

> [!TIP]
> Retrying an entire workflow throws away all accumulated state and starts from scratch. Handle failures through activity retries and workflow compensation logic instead.

When an activity retries, you re-execute a single operation. When a workflow retries, you throw away everything -- every completed activity, every decision, every side effect -- and start from item 1. If your workflow processed 99 of 100 items before failing, a workflow retry discards all that progress.

This wastes work and requires reasoning about the idempotency of the entire workflow; not just each activity and isolation.
