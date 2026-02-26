# Not Monitoring Schedule-To-Start Latency

> [!TIP]
> * [Schedule-To-Start Latency](terms/schedule-to-start-latency.md) (STSL) measures the time between a task being scheduled and a [worker](terms/worker.md) picking it up.
> * High STSL means workers can't keep up with the task rate -- you need more workers, faster workers, or fewer tasks.
> * This is one of the most important operational metrics for Temporal and directly impacts end-to-end workflow latency.

## What?

Schedule-To-Start Latency is the delay between the moment Temporal schedules a task ([workflow task](terms/workflow-task.md) or activity task) and the moment a worker actually starts executing it. In a healthy system, this latency is near zero because a worker is almost always available to pick up new tasks immediately. When STSL starts climbing, it means tasks are sitting in a queue waiting for a worker to become available.

Many teams deploy Temporal without monitoring this metric, then wonder why their workflows feel slow even though individual activities complete quickly. The bottleneck isn't execution time -- it's time spent waiting in the queue.

## Why?

STSL is the single best indicator of worker capacity health. Unlike end-to-end workflow latency, which combines many factors, STSL isolates exactly one thing: are your workers keeping up with demand?

When STSL is high:
- **End-to-end latency degrades** because every task in the workflow pays the queueing penalty. A workflow with 10 sequential activities, each waiting 5 seconds in the queue, adds 50 seconds of pure overhead.
- **Timeouts can fire prematurely**. The `ScheduleToStart` timeout on activities counts from when the task is scheduled, not when the worker picks it up. High STSL eats into this budget.
- **Cascading effects** can occur. As tasks pile up, workers may start timing out on [heartbeats](terms/heartbeat.md) for running activities, causing retries that create even more tasks in the queue.

## How?

1. **Monitor the `temporal_activity_schedule_to_start_latency` and `temporal_workflow_task_schedule_to_start_latency` metrics** from your workers' SDK metrics. Set up alerts when STSL exceeds acceptable thresholds (e.g. p99 above 1 second).

2. **Diagnose the root cause** when STSL rises:
   - Too few workers: scale up the number of worker instances.
   - Workers too slow: individual activity executions are taking too long, holding up worker slots. Profile and optimize hot activities, or increase `MaxConcurrentActivities`.
   - Too few pollers per worker: increase the poller count if workers are idle but not polling fast enough.
   - [Task queue](terms/task-queue.md) imbalance: if you use multiple task queues, check whether load is unevenly distributed.

3. **Set up dashboards** that show STSL alongside worker count and activity execution latency so you can correlate spikes with deployment events or traffic changes.

4. **Monitor STSL alongside [sync match rate](not-monitoring-sync-match-rate.md)** for a complete picture of task delivery health.
