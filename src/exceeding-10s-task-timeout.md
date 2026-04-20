# Exceeding the 10-Second Workflow Task Timeout

> [!TIP]
> [Workflow tasks](terms/workflow-task.md) must complete within 10 seconds by default. Exceeding this causes a livelock: the task times out, gets retried, [replays](terms/replay.md), hits the same bottleneck, and times out again.

A workflow task is the unit of work a [worker](terms/worker.md) processes when executing workflow code. Each time, the worker replays the full [history](terms/event-history.md) and then executes new code until the next yield point. If this takes more than 10 seconds, the server reschedules the task and the cycle repeats. You'll see `WorkflowTaskTimedOut` events accumulating in history.

Three common causes: expensive computation in workflow code (JSON parsing of large [payloads](terms/payload.md), data transformations), replaying a large history (tens of thousands of events), or scheduling thousands of activities or [child workflows](terms/child-workflow.md) without yielding.

Solutions:
- Keep workflow code lightweight -- move computation to activities
- Use [ContinueAsNew](terms/continue-as-new.md) to bound history size and reduce replay time
- Batch large fan-outs: schedule a batch, yield, schedule the next batch
- Ensure workflow caching is effective so subsequent tasks skip replay
- Increase `WorkflowTaskTimeout` only as a last resort -- it's a band-aid for a deeper design issue

See also: [Performing Expensive Computation in Workflow Code](performing-expensive-computation-in-workflow-code.md), [Overflowing Workflow History Size](overflowing-workflow-history-size.md).
