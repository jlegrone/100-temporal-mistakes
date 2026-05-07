# Querying Closed Workflows

<!-- TODO: Write an interceptor to add query result to memo attribute before workflow completion? -->

> [!TIP]
> [Queries](terms/queries.md) on closed workflows require full [replay](terms/replay.md) of the [history](terms/event-history.md), which fails if the workflow code has changed incompatibly since the workflow ran. Store information you need to read from closed workflows in the return value or memo attributes of the workflow instead.

Queries dispatch to a [worker](terms/worker.md) which replays the workflow's history to reconstruct state, then runs the query handler. For closed workflows, this means replaying the entire history from scratch. This makes querying both slow and brittle, since the old workflow histories may not be replay compatible with the current version of the worker.

This becomes increasingly fragile as the codebase evolves. A query that worked yesterday might break today after a deployment.

Instead: Use custom [search attributes](terms/search-attributes.md) for simple state (status, progress, business identifiers) -- they're stored on the server and queryable via [visibility](terms/visibility.md) APIs without replay.

<!-- TODO: Link to worker controller & sunset strategy? -->
