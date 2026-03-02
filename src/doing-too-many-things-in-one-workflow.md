# Doing Too Many Things in One Workflow

> [!TIP]
> * A single workflow that orchestrates too many concerns leads to large [histories](terms/event-history.md), shard contention, complex [versioning](terms/versioning.md), and a single point of failure.
> * Scale out (many small workflows) rather than scale up (one giant workflow).
> * Use [child workflows](terms/child-workflow.md) or separate top-level workflows for independent concerns.

## What?

A common anti-pattern is building a "god workflow" that handles every aspect of a business process in a single workflow execution. For example, an order processing workflow that handles payment, inventory, shipping, notifications, analytics, and customer support ticket creation all in one place.

While it may seem convenient to have everything in one workflow, this approach breaks down as the system grows.

## Why?

**Large histories**: Every activity, timer, and child workflow adds events to the workflow history. A workflow that does everything accumulates events quickly, leading to longer [replay](terms/replay.md) times and eventually [overflowing the history size limit](<overflowing-workflow-history-size.md>).

**[Workflow lock contention](<workflow-lock-contention-due-to-concurrent-updates.md>)**: When a single workflow handles many concerns, it often receives concurrent updates ([signals](terms/signals.md), activity completions, [queries](terms/queries.md)) that compete for the workflow lock. This creates contention and increases end-to-end latency.

**Complex versioning**: When you need to change one aspect of the workflow (say, the notification logic), you must version the entire monolithic workflow. With separate workflows, you only version the one that changed. [ContinueAsNew](terms/continue-as-new.md) also becomes harder to implement when the workflow carries a large amount of diverse state.

**Single point of failure**: If the workflow gets stuck, [terminates](terms/terminate.md), or hits a bug, every concern it handles is affected. Independent workflows isolate failures.

**Team scalability**: When multiple teams need to modify the same workflow, coordination overhead increases and merge conflicts become frequent.

## How?

**Decompose by concern**: Identify independent concerns and give each its own workflow. The order workflow handles order state. A separate notification workflow handles notifications. A separate analytics workflow handles event tracking.

**Use child workflows for sub-processes**: When a concern is logically subordinate to a parent process, model it as a child workflow. For example, a payment workflow might spawn a child workflow for each retry strategy or payment method.

**Coordinate with signals or by starting new workflows**: Independent workflows can be started from activities or from the parent workflow. When they need to communicate, use signals or store shared state externally.

**Keep workflows focused**: A good rule of thumb is that a workflow should represent a single entity or a single process with a clear lifecycle. If you find yourself saying "this workflow also handles...", it is probably time to split.

The goal is not to have the smallest possible workflows, but to have workflows with clear boundaries and manageable complexity. Many small workflows is how Temporal is designed to scale.
