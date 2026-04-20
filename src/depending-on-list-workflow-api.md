# Depending on the ListWorkflow API for Application Logic

> [!TIP]
> [Visibility](terms/visibility.md) APIs are eventually consistent, rate-limited, and may return incomplete results. Use deterministic [workflow IDs](terms/workflow-id.md) or external references instead of querying for workflows at runtime.

Teams sometimes use `ListWorkflow` or other visibility APIs to find workflows to [signal](terms/signals.md), check whether a workflow exists, or coordinate between services. These APIs excel at operational tooling and debugging, but they're backed by an eventually consistent store. A just-started workflow may not appear in query results for seconds or minutes. Under load, queries get throttled. Paginated results with concurrent workflow changes mean you can never be sure you've seen all matching workflows.

Instead: derive workflow IDs deterministically from domain data (e.g., `order-{orderID}`) so you can interact with them directly by ID. When you start a workflow, store its ID in your own database if you need to find it later. Use parent-child relationships for workflow-to-workflow coordination. Reserve `ListWorkflow` for operational dashboards and debugging.
