# Depending on the ListWorkflow API for Application Logic

> [!TIP]
> [Visibility](terms/visibility.md) APIs are eventually consistent, rate-limited, and may return incomplete results. Use deterministic [workflow IDs](terms/workflow-id.md) or external databases instead of querying for workflows at runtime.

<!-- TODO: Revisit this, it needs more work and fact checking. -->
Teams sometimes use the `ListWorkflow` API to drive external application interfaces. The Temporal visibility API is designed for , but they're backed by an eventually consistent store. A just-started workflow may not appear in query results for seconds or minutes. Under load, queries get throttled. Paginated results with concurrent workflow changes mean you can never be sure you've seen all matching workflows.

Instead: derive workflow IDs deterministically from domain data (e.g., `order-{orderID}`) so you can interact with them directly by ID.

<!-- TODO: If workflows are tied to an entity in your application, then write records to a database you own from your workflows and then query that database directly when implementing your public API. -->
<!-- TODO: Include a "payments" example, where a user calls an API which triggers a SendMoney workflow, the workflow writes the transaction record to a database, and then the same user calls a ListTransactions API which queries the database directly. -->
