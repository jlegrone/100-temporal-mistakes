# Schedule-to-Start Latency

Schedule-to-Start Latency (STSL) is the time between when a task (workflow task or activity task) is scheduled by the server and when a worker picks it up for execution. It is one of the most important operational metrics for Temporal deployments.

High STSL indicates that workers cannot keep up with the incoming task rate. This can be caused by insufficient worker capacity, too few pollers, slow activity execution blocking the worker's slots, or workers not listening on the correct task queue. STSL should be monitored alongside the sync match rate for a complete picture of task delivery health.

## Related

- [Not Monitoring STSL](../not-monitoring-stsl.md)
- [Not Monitoring Sync Match Rate](../not-monitoring-sync-match-rate.md)
- [Starting Workflows on Wrong Task Queue](../starting-workflows-on-wrong-task-queue.md)
- [Not Enabling Autotuning](../not-enabling-autotuning.md)
- [Task Queue](task-queue.md)
- [Worker](worker.md)
- [Activity Task](activity-task.md)
- [Workflow Task](workflow-task.md)
