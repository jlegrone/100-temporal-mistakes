# Exceeding the 10-Second Workflow Task Timeout

> [!TIP]
> [Workflow tasks](terms/workflow-task.md) must complete within 10 seconds by default. Exceeding this may cause a livelock: the task times out, gets retried, [replays](terms/replay.md), hits the same bottleneck, and times out again.

A workflow task is the unit of work a [worker](terms/worker.md) processes when executing workflow code. A workflow task executes until the next yield point. If this takes more than 10 seconds, the server reschedules the task and the cycle repeats. You'll see `WorkflowTaskTimedOut` events accumulating in history.

<!-- TODO: Refactor this paragraph to list format -->
Common causes: expensive computation in workflow code (JSON parsing of large [payloads](terms/payload.md), data transformations), replaying a large history (tens of thousands of events), or scheduling thousands of activities or [child workflows](terms/child-workflow.md) without yielding, or slow dataconverter encode/decode.

Solutions:
- Keep workflow code lightweight -- move computation to activities
- Use [ContinueAsNew](terms/continue-as-new.md) to bound history size and reduce replay time
- Batch large fan-outs: schedule a batch, yield, schedule the next batch
- Ensure [workflow caching](https://docs.temporal.io/develop/worker-performance#workflow-cache-tuning) is effective so subsequent tasks skip replay -- if `sticky_cache_size` is consistently at the configured max, the cache is full and workflows are being evicted -- increase the cache size if workers have free RAM
- Increase `WorkflowTaskTimeout` only as a last resort -- it's a band-aid for a deeper design issue

See also: [Performing Expensive Computation in Workflow Code](../performing_expensive_computation_in_workflow_code/), [Overflowing Workflow History Length](overflowing-workflow-history-length.md)
<!-- TODO: Also link to returning too much data from activities -->
