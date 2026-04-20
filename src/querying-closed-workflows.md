# Querying Closed Workflows

> [!TIP]
> [Queries](terms/queries.md) on closed workflows require full [replay](terms/replay.md) of the [history](terms/event-history.md), which fails if the workflow code has changed incompatibly since the workflow ran.

Queries dispatch to a [worker](terms/worker.md) which replays the workflow's history to reconstruct state, then runs the query handler. For closed workflows, this means replaying the entire history from scratch. If the code has changed in a [non-deterministic](terms/non-determinism.md) way (activities added/removed/reordered without proper [versioning](terms/versioning.md)), replay fails and the query returns an error.

This becomes increasingly fragile as the codebase evolves. A query that worked yesterday might break today after a deployment.

Instead: emit query-relevant data to an external store during workflow execution. Use [search attributes](terms/search-attributes.md) for simple state (status, progress, business identifiers) -- they're stored on the server and queryable via [visibility](terms/visibility.md) APIs without replay. If you must query closed workflows, maintain strictly backwards-compatible code and set [namespace](terms/namespace.md) retention policies to limit how far back queries can reach.
