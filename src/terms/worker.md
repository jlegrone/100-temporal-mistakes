# Worker

A worker is a process that hosts and executes workflow and activity code. Workers connect to the Temporal server by polling one or more task queues for tasks. When a worker receives a workflow task, it replays the workflow code to reconstruct state and process new events. When it receives an activity task, it executes the activity function.

Workers are deployed and scaled independently of the Temporal server. Multiple workers can poll the same task queue for horizontal scaling. Workers are stateless from Temporal's perspective -- any worker polling the right task queue can pick up any task, though workflow caching can improve performance by keeping workflow state in memory between tasks.

## Related

- [Not Draining Activity Tasks Before Shutdown](../not-draining-activity-tasks-before-shutdown.md)
- [Not Enabling Autotuning](../not-enabling-autotuning.md)
- [Not Monitoring STSL](../not-monitoring-stsl.md)
- [Starting Workflows on Wrong Task Queue](../starting-workflows-on-wrong-task-queue.md)
- [Task Queue](task-queue.md)
- [Workflow Task](workflow-task.md)
- [Activity Task](activity-task.md)
- [Replay](replay.md)
