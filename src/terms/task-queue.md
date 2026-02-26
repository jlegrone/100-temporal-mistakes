# Task Queue

A task queue is a named queue that connects workflow/activity starters to the workers that execute them. When a workflow or activity is scheduled, a task is placed on the specified task queue. Workers poll specific task queues to receive work. Task queues are created implicitly when a worker starts polling or a workflow/activity is scheduled -- there is no explicit creation step.

Task queues enable several architectural patterns: routing work to specific worker pools (e.g., workers with GPU access), scaling workers independently for different workloads, and deploying different versions of workers. A common source of errors is starting a workflow on a task queue that no worker is listening on, which causes the workflow to wait indefinitely.

## Related

- [Starting Workflows on Wrong Task Queue](../starting-workflows-on-wrong-task-queue.md)
- [Not Monitoring STSL](../not-monitoring-stsl.md)
- [Not Monitoring Sync Match Rate](../not-monitoring-sync-match-rate.md)
- [Worker](worker.md)
- [Workflow Task](workflow-task.md)
- [Activity Task](activity-task.md)
