# Querying Closed Workflows

> [!TIP]
> * [Queries](terms/queries.md) execute against a workflow's current state by replaying its [history](terms/event-history.md) on a [worker](terms/worker.md).
> * For closed workflows, this requires full [replay](terms/replay.md) of the entire history, which fails if the workflow code has changed incompatibly.
> * Be cautious about querying old completed workflows -- consider storing query-relevant data externally instead.

## What?

[Queries](terms/queries.md) are a read-only mechanism to inspect a workflow's current state. Under the hood, the query is dispatched to a worker that has the workflow code registered. The worker [replays](terms/replay.md) the workflow's history to reconstruct its state, then executes the query handler against that state.

For running workflows that are already cached in a worker's memory, this is fast and straightforward. For closed (completed, failed, cancelled, or terminated) workflows, the worker must replay the entire history from scratch to answer the query.

## Why?

The problem arises when the workflow code has changed between when the workflow ran and when you're querying it. [Replay](terms/replay.md) re-executes the workflow code against the recorded history. If the code has changed in a non-deterministic way (e.g., an activity was added, removed, or reordered without proper [versioning](terms/versioning.md)), replay fails with a [non-determinism](terms/non-determinism.md) error and the query returns an error.

This means that querying old completed workflows becomes increasingly fragile as the codebase evolves. A query that worked fine yesterday might break today because someone deployed a code change that isn't backwards-compatible with the old workflow's history.

The risk is higher for:
- **Long-lived workflows** that completed months ago, where the code may have changed significantly.
- **Workflows that weren't properly versioned**, making replay impossible after code changes.
- **High-throughput systems** where replaying large histories for queries adds unexpected load to workers.

## How?

1. **Store query-relevant data externally.** If you need to access workflow state after completion, emit the relevant data to an external store (database, search index) during workflow execution. Query the external store instead of the closed workflow.

2. **Use [search attributes](terms/search-attributes.md).** Temporal search attributes are stored on the server and queryable via [visibility](terms/visibility.md) APIs without replaying the workflow. For simple state like status, progress percentage, or business identifiers, search attributes are a better fit than queries.

3. **Maintain backwards-compatible code.** If you must query closed workflows, ensure your workflow code changes are always backwards-compatible using the [versioning](terms/versioning.md) APIs. This requires discipline and increases code complexity over time.

4. **Set retention policies.** Configure [namespace](terms/namespace.md) retention periods to automatically clean up old workflow histories. This limits how far back queries can reach, which also limits how much versioning baggage you need to carry.
