# Activity Task

An activity task represents a unit of work scheduled by a workflow to be executed by a worker. When a workflow calls `ExecuteActivity`, the server creates an activity task on the specified task queue. A worker picks up the task, executes the activity function, and reports the result back to the server, which records it in the workflow's event history.

Activity tasks have their own set of timeouts (schedule-to-start, start-to-close, schedule-to-close, heartbeat) and retry policies. Unlike workflow tasks, activity tasks can be long-running (hours or days) as long as they heartbeat regularly.

## Related

- [Not Draining Activity Tasks Before Shutdown](../not_draining_activity_tasks_before_shutdown/)
- [Not Monitoring STSL](../not-monitoring-stsl.md)
- [Preventing Activity Retries](../preventing-activity-retries.md)
- [Over-Using Activities](../over-using-activities.md)
- [Workflow Task](workflow-task.md)
- [Task Queue](task-queue.md)
- [Worker](worker.md)
- [Heartbeat](heartbeat.md)
- [Retry Policy](retry-policy.md)
