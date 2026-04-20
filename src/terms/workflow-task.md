# Workflow Task

A workflow task is the internal unit of work that a worker processes to advance a workflow's execution. When a workflow needs to make progress (e.g., an activity completed, a signal was received, a timer fired), the Temporal server creates a workflow task and places it on the workflow's task queue. The worker picks up the task, replays the workflow code to reconstruct state, processes the new event, and returns the resulting commands (schedule activity, start timer, complete workflow, etc.) back to the server.

Workflow tasks have a default timeout of 10 seconds. If a workflow task takes longer than this (due to expensive computation, large history replay, or scheduling too many operations), it times out and is retried, potentially causing a livelock.

## Related

- [Exceeding 10s Task Timeout](../exceeding-10s-task-timeout.md)
- [Performing Expensive Computation in Workflow Code](../performing_expensive_computation_in_workflow_code/)
- [Using Local Activities](../using_local_activities/)
- [Activity Task](activity-task.md)
- [Replay](replay.md)
- [Event History](event-history.md)
- [Task Queue](task-queue.md)
