# Depending on the ListWorkflow API for Application Logic

> [!TIP]
> * The ListWorkflow API and visibility APIs are designed for observability and debugging, not for driving application logic.
> * These APIs have eventual consistency, rate limits, and may not return complete results.
> * Use deterministic workflow IDs or store references externally instead of querying for workflows at runtime.

## What?

Teams sometimes reach for the `ListWorkflow` API (or other visibility APIs like `ListOpenWorkflowExecutions`, `ListClosedWorkflowExecutions`) to implement application logic. Common examples include:

- Querying for workflows to send [signals](terms/signals.md) to
- Checking whether a workflow exists before making business decisions
- Using visibility queries as a coordination mechanism between services
- Building dashboards that feed back into automated decision-making

While these APIs are excellent for operational tooling, debugging, and observability, using them as a foundation for application logic introduces subtle but serious reliability issues.

## Why?

Visibility APIs are backed by an eventually consistent store. When a workflow is started, there is a non-trivial delay before it appears in visibility query results. Similarly, when a workflow completes, the status update propagates asynchronously. This means:

1. **Eventual consistency**: A workflow that was just started may not show up in a `ListWorkflow` query for seconds or even minutes depending on your [temporal server backend](terms/temporal-server-backend.md) configuration and load. You might miss workflows or see stale state.
2. **Rate limits**: Visibility APIs are rate-limited. Under high load, your queries may be throttled, causing your application logic to stall or miss data.
3. **Incomplete results**: Paginated results combined with concurrent workflow creation and completion mean you can never be sure you've seen all matching workflows at a given point in time.
4. **Coupling to server internals**: The behavior and performance of visibility APIs varies across Temporal server versions and backend storage configurations (Elasticsearch vs SQL).

Building on these APIs for correctness-critical logic means your application inherits all of these limitations.

## Solution

Instead of querying for workflows, use one of these approaches:

1. **Deterministic workflow IDs**: If you need to interact with a specific workflow, derive its ID deterministically from your domain data (e.g., `order-{orderID}`, `user-{userID}-subscription`). Then you can signal, query, or describe it directly by ID without searching.
2. **Store references externally**: When you start a workflow, store its ID in your own database. Query your database instead of the Temporal visibility store when you need to find workflows.
3. **Use parent-child relationships**: If one workflow needs to coordinate with others, use child workflows. The parent workflow maintains direct references to its children through Temporal's built-in parent-child mechanism.
4. **Reverse the dependency**: Instead of searching for workflows to signal, have the workflows themselves reach out (via activities) when they need to coordinate, or use a well-known workflow ID as a rendezvous point.

Reserve the `ListWorkflow` API for what it was designed for: building operational dashboards, debugging production issues, and monitoring workflow health.
