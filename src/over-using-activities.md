# Over-Using Activities

> **TL;DR**
> Every activity creates [history](terms/event-history.md) events, requires serialization, and adds a round-trip to the server. Batch related work into fewer, coarser-grained activities instead of creating one per tiny operation.

A common mistake is treating activities like function calls -- creating separate activities for "get user", "validate fields", "format response", and "log result" instead of a single "process user" activity. This comes from applying single-responsibility principles without accounting for the cost model of activities.

Each activity generates at least three history events, serializes its inputs and outputs, and requires a network round-trip between the [worker](terms/worker.md) and server. Hundreds of fine-grained activities accumulate thousands of events, increasing [replay](terms/replay.md) time and pushing toward the [history length limit](overflowing-workflow-history-length.md). Chaining 20 sequential activities at 5ms overhead each adds 100ms of pure infrastructure latency.

Group related operations into a single activity. Use activities at the boundary between your workflow and external systems. Ask: "Does this operation need its own [retry policy](terms/retry-policy.md), timeout, or [heartbeat](terms/heartbeat.md)?" If not, it probably doesn't need to be a separate activity. But don't go too far -- a single activity that runs for 30 minutes and does everything can't be partially retried and doesn't report progress.
