# Sessions

Sessions are a Temporal feature that routes multiple activity executions to the same worker. This is useful when activities need access to local resources on a specific machine, such as files on disk, GPU devices, or in-memory caches. A session is created on a worker and all subsequent activities within that session are guaranteed to run on the same worker, until the session is released or times out.

## Related

- [Overflowing maximum individual payload size](../overflowing-maximum-individual-payload-size.md)
- [Worker](worker.md)
- [Activity Task](activity-task.md)
- [Task Queue](task-queue.md)
