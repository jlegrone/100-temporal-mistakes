# Unnecessary Usage of Workflows

> [!TIP]
> Workflow creation, history persistence, and replay are not free. Use workflows when you genuinely need durability, reliable retries, or coordination across failures -- not for every operation in your system.

When teams adopt Temporal, there's a natural temptation to route everything through workflows. But using workflows comes with a degree of operational and runtime overhead. For operations that don't need durability, this overhead is pure cost with no benefit.

Before reaching for a workflow, ask: does this operation need to survive process crashes? Does it need to be retried reliably if it fails? Does it span multiple services or steps that need coordination? If the caller can be responsible for retrying the request, or the failure is acceptable, a workflow adds no value. But even a fast operation benefits from a workflow if it must succeed despite crashes or restarts.

Good candidates for workflows include multi-step business processes, operations requiring reliable retries, scheduled jobs, and processes that need human approval steps.
