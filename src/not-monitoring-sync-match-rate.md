# Not Monitoring Sync Match Rate

> [!TIP]
> * Sync match occurs when a task is immediately dispatched to a waiting worker poller without being persisted to the database.
> * A high sync match rate means low task delivery latency and reduced load on the persistence layer.
> * A dropping sync match rate is an early warning signal that workers are falling behind -- often before Schedule-To-Start Latency visibly spikes.

## What?

When Temporal's matching service receives a new task, it first checks if there is already a worker polling and waiting for work on that task queue. If there is, the task is handed directly to that worker without ever being written to the persistence store. This is called a **sync match** and it is the fast path for task delivery.

When no worker is immediately available, the task must be persisted to the database and will be picked up later when a worker eventually polls. This is an **async match** -- it works, but it is slower because it involves a write and a subsequent read from the persistence layer.

The sync match rate is the percentage of tasks that take the fast path. Many teams never look at this metric, missing an important signal about the health of their task delivery pipeline.

## Why?

Sync match rate matters for two reasons:

- **Latency**: A sync-matched task skips the persistence round-trip entirely. This can shave tens of milliseconds off each task delivery, which compounds across a workflow with many sequential tasks.
- **Early warning**: Sync match rate tends to degrade before [Schedule-To-Start Latency](not-monitoring-stsl.md) gets bad enough to trigger alerts. If you notice sync match rate dropping from 95% to 70%, your system is telling you that workers are starting to saturate -- even if STSL hasn't visibly spiked yet. This gives you time to react before users notice.

A healthy Temporal deployment typically has a sync match rate above 90% for its primary task queues. If yours is significantly lower, something is off.

## How?

1. **Monitor the `temporal_matching_sync_match` metric** exposed by the Temporal server (not the SDK). Track it per task queue so you can identify which queues are under pressure.

2. **Set up alerts** when sync match rate drops below a threshold (e.g. below 80%) for a sustained period. This is your early warning to scale workers before STSL degrades.

3. **Diagnose drops** by checking:
   - **Not enough pollers**: Each worker maintains a pool of long-poll requests to the server. If the poller count is too low relative to the task rate, there may not always be a poller waiting when a task arrives. Increase the poller count on workers.
   - **Worker saturation**: Workers that are fully busy executing activities or workflow tasks won't have idle poller slots. Scale up the number of worker instances.
   - **Bursty traffic**: If tasks arrive in bursts, even a well-provisioned system may see temporary dips in sync match rate. Smoothing out task scheduling can help.

4. **Use sync match rate and [STSL](not-monitoring-stsl.md) together**: sync match rate is the leading indicator, STSL is the lagging one. If sync match rate drops but STSL stays low, you have time to act. If both degrade, you're already behind.
