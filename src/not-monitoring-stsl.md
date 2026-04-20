# Not Monitoring Schedule-To-Start Latency

> [!TIP]
> [Schedule-To-Start Latency](terms/schedule-to-start-latency.md) is the single best indicator of worker capacity health. If tasks are sitting in a queue waiting for a worker, your end-to-end workflow latency will suffer regardless of how fast individual activities execute.

Schedule-To-Start Latency (STSL) measures the delay between when Temporal schedules a task ([workflow task](terms/workflow-task.md) or activity task) and when a [worker](terms/worker.md) starts executing it. In a healthy system, this latency is near zero because a worker is almost always available to pick up new tasks immediately. When STSL climbs, tasks are sitting in a queue waiting. A workflow with 10 sequential activities, each waiting 5 seconds in the queue, adds 50 seconds of pure overhead. High STSL also eats into `ScheduleToStart` timeout budgets, and as tasks pile up, workers start timing out on [heartbeats](terms/heartbeat.md) for running activities, causing retries that create even more tasks.

Monitor the `temporal_activity_schedule_to_start_latency` and `temporal_workflow_task_schedule_to_start_latency` metrics from your workers' SDK metrics, and set up alerts when STSL exceeds acceptable thresholds (e.g., p99 above 1 second). When STSL rises, diagnose whether the issue is too few workers, workers that are too slow (profile and optimize hot activities or increase `MaxConcurrentActivities`), too few pollers per worker, or a [task queue](terms/task-queue.md) imbalance. Set up dashboards showing STSL alongside worker count and activity execution latency so you can correlate spikes with deployment events or traffic changes.

See also: [Not Monitoring Sync Match Rate](not-monitoring-sync-match-rate.md) for a leading indicator that often degrades before STSL visibly spikes.
