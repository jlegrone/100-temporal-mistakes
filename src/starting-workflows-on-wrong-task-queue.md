# Starting Workflows on the Wrong Task Queue

> [!TIP]
> A [task queue](terms/task-queue.md) mismatch between starters and [workers](terms/worker.md) causes workflows to wait indefinitely with no error -- Temporal treats missing workers as temporary by design.

Task queues route workflow and activity tasks to workers. If the task queue name in `StartWorkflow` doesn't match what workers poll, the task sits in the queue forever. The `StartWorkflow` call succeeds (returning a valid [workflow ID](terms/workflow-id.md)), but the workflow never makes progress. Common causes: typos (`"my-task-queue"` vs `"my_task_queue"`), inconsistent configuration between services, or deploying a new workflow without registering workers on the correct queue.

Use shared constants for task queue names -- never hardcode strings in multiple places. Monitor [Schedule-To-Start latency](terms/schedule-to-start-latency.md): if STSL exceeds a few seconds, something is wrong. Use the Temporal UI or CLI to check whether pollers are registered on a given task queue.
