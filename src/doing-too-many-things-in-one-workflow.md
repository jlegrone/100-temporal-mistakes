# Doing Too Many Things in One Workflow

> [!TIP]
> A "god workflow" that handles every concern leads to large [histories](terms/event-history.md), [lock contention](workflow-lock-contention-due-to-concurrent-updates.md), complex [versioning](terms/versioning.md), and a single point of failure. Scale out across many workflows, not up within one.

A common anti-pattern is building a single workflow that handles every aspect of a business process -- payment, inventory, shipping, notifications, analytics, and support tickets all in one place. Every activity, timer, and [child workflow](terms/child-workflow.md) adds events to the history. The workflow accumulates events quickly, leading to longer [replay](terms/replay.md) times, eventual [history overflow](overflowing-workflow-history-length.md), and [workflow lock contention](workflow-lock-contention-due-to-concurrent-updates.md) as concurrent updates compete.

Versioning also suffers: changing the notification logic requires versioning the entire monolithic workflow. If the workflow gets stuck or [terminates](terms/terminate.md), every concern is affected. And when multiple teams modify the same workflow, coordination overhead grows.

Decompose by concern: the order workflow handles order state, a separate workflow handles notifications, another handles analytics. Use child workflows for logically subordinate sub-processes. A good rule of thumb: a workflow should represent a single entity or process with a clear lifecycle. If you find yourself saying "this workflow also handles...", it's time to split. Many small workflows is how Temporal is designed to scale.
