# Using Workflow Retries

> [!TIP]
> Retrying an entire workflow throws away all accumulated state and starts from scratch -- the opposite of what makes Temporal useful. Handle failures internally through activity retries and compensation logic.

When an activity retries, you re-execute a single operation. When a workflow retries, you throw away everything -- every completed activity, every decision, every side effect -- and start from item 1. If your workflow processed 99 of 100 items before failing, a workflow retry discards all that progress.

This wastes work and risks duplicating non-[idempotent](terms/idempotency.md) side effects (payments, notifications, resource creation). If the failure was a bug in workflow logic rather than a transient issue, retrying just fails the same way again.

Remove workflow-level [retry policies](terms/retry-policy.md) from most workflows. Push retry logic to activities, each with its own policy tuned to the operation. Implement compensation for partial failures (the Saga pattern). Reserve workflow retries for truly simple cases: a workflow that calls a single activity with no meaningful state to preserve.
