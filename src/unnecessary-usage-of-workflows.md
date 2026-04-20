# Unnecessary Usage of Workflows

> [!TIP]
> Workflow creation, history persistence, and replay are not free. Use workflows when you genuinely need durability, reliable retries, or coordination across failures -- not for every operation in your system.

## What?

When teams adopt Temporal, there is a natural temptation to route everything through workflows. After all, if workflows give you retries, observability, and durability, why not use them for everything?

But Temporal workflows come with overhead. Creating a workflow means persisting a [history](terms/event-history.md), scheduling tasks, consuming server resources, and replaying state. For operations that don't need to be durably retried, this overhead is pure cost with no benefit.

## Why?

**Latency**: Starting a workflow, scheduling an activity, and waiting for its completion adds latency compared to a direct function call or API request. For user-facing synchronous operations where response time matters, this overhead can be significant.

**Resource consumption**: Every workflow execution consumes server resources: database storage for the history, shard capacity, and [task queue](terms/task-queue.md) throughput. Routing trivial operations through Temporal wastes these resources and impacts the performance of workflows that actually need them.

**Complexity**: Wrapping simple operations in workflows adds code (workflow definitions, activity definitions, [worker](terms/worker.md) setup) without adding value. It makes the codebase harder to navigate and increases the surface area for bugs.

**Operational burden**: More workflows means more to monitor, more history to retain, and more potential for issues like task queue backlogs. The operational cost scales with the number of workflow executions.

## How?

Before reaching for a workflow, ask:
- **Does this operation need to survive process crashes?** If a failure just means the user retries their request, you probably don't need a workflow.
- **Does this operation need to be retried reliably if it fails?** If the caller can simply retry the request, or the failure is acceptable, a workflow adds no value. But even a fast operation benefits from a workflow if it must succeed despite crashes or restarts.
- **Does this operation span multiple services or steps that need coordination?** If it is a single database write or API call, a workflow adds no value.
- **Do you need to track the state of this operation over time?** If you don't need to query progress or wait for completion asynchronously, a workflow is overkill.

Good candidates for workflows include: multi-step business processes, long-running operations, operations requiring reliable retries across external systems, scheduled or recurring jobs, and processes that need human approval steps.

Poor candidates include: simple CRUD endpoints, input validation, cache lookups, logging, metrics emission, and any operation where the caller is synchronously waiting for an immediate response.

Temporal works best as a control plane for orchestrating complex, long-running, or failure-prone operations. Treat it as such, and keep simple operations simple.
