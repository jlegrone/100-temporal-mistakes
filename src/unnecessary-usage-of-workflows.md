# Unnecessary Usage of Workflows

> [!TIP]
> Workflow creation, history persistence, and replay are not free. Use workflows when you genuinely need durability, reliable retries, or coordination across failures -- not for every operation in your system.

When teams adopt Temporal, there's a natural temptation to route everything through workflows. But creating a workflow means persisting a [history](terms/event-history.md), scheduling tasks, and consuming server resources. For operations that don't need durable retries, this overhead -- in latency, resource consumption, code complexity, and operational burden -- is pure cost with no benefit.

Before reaching for a workflow, ask: does this operation need to survive process crashes? Does it need to be retried reliably if it fails? Does it span multiple services or steps that need coordination? If the caller can simply retry the request, or the failure is acceptable, a workflow adds no value. But even a fast operation benefits from a workflow if it must succeed despite crashes or restarts.

Good candidates for workflows include multi-step business processes, operations requiring reliable retries across external systems, scheduled or recurring jobs, and processes that need human approval steps. Poor candidates include input validation, cache lookups, logging, and metrics emission.

See also: [Not Understanding Why You're Using Temporal](not-understanding-why-youre-using-temporal.md).
