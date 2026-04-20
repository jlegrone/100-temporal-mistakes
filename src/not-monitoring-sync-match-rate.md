# Not Monitoring Sync Match Rate

> [!TIP]
> A dropping sync match rate is an early warning that workers are falling behind -- often before [Schedule-To-Start Latency](terms/schedule-to-start-latency.md) visibly spikes. Monitor it to catch capacity problems early.

When Temporal's matching service receives a new task, it checks whether a [worker](terms/worker.md) is already polling and waiting for work on that [task queue](terms/task-queue.md). If so, the task goes directly to that worker without being written to the persistence store -- this is a **sync match**, the fast path. When no worker is immediately available, the task is persisted and picked up later -- an **async match** that is slower because it involves a database round-trip. A healthy Temporal deployment typically has a sync match rate above 90% for its primary task queues.

Sync match rate matters for both latency and early warning. A sync-matched task skips the persistence round-trip, shaving tens of milliseconds off each delivery, which compounds across workflows with many sequential tasks. More importantly, sync match rate tends to degrade before [STSL](not-monitoring-stsl.md) gets bad enough to trigger alerts, giving you time to react before users notice.

Monitor the `temporal_matching_sync_match` metric exposed by the Temporal server (not the SDK), tracked per task queue. Set up alerts when sync match rate drops below a threshold (e.g., below 80%) for a sustained period. Diagnose drops by checking poller count (increase if workers are idle but not polling fast enough), worker saturation (scale up instances if workers are fully busy), and traffic burstiness (smooth out task scheduling if tasks arrive in bursts). Use sync match rate and STSL together: sync match rate is the leading indicator, STSL is the lagging one.

See also: [Not Monitoring Schedule-To-Start Latency](not-monitoring-stsl.md).
