# Doing Too Many Things in One Workflow

> [!TIP]
> A "monolithic workflow" that handles every concern leads to large [histories](terms/event-history.md), [lock contention](workflow-lock-contention-due-to-concurrent-updates.md) and complex [versioning](terms/versioning.md). Decompose complex tasks into child workflows.

A common anti-pattern is building a single workflow that handles every aspect of a business process -- payment, inventory, shipping, notifications, analytics, and support tickets all in one place.

Decompose by concern: the order workflow handles order state, a separate workflow handles notifications, another handles analytics. Use child workflows to separate concerns. A good rule of thumb: a workflow should represent a single entity or process with a clear lifecycle.
