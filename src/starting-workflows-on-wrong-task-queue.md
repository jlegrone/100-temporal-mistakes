# Starting Workflows on the Wrong Task Queue

> [!TIP]
> * [Task queues](terms/task-queue.md) are the link between workflow/activity starters and the [workers](terms/worker.md) that execute them. A mismatch means tasks sit indefinitely with no one to pick them up.
> * There are no errors when this happens -- the workflow simply waits forever, making it hard to notice without monitoring.
> * Always validate task queue names through shared constants or configuration, and monitor [Schedule-To-Start latency](terms/schedule-to-start-latency.md) to catch mismatches early.

## What?

Task queues are named queues that Temporal uses to route [workflow tasks](terms/workflow-task.md) and activity tasks to the correct workers. When you start a workflow, you specify which task queue it should be placed on. Workers poll specific task queues for work. If these don't match -- due to a typo, a misconfiguration, or a deployment targeting a different service -- the workflow task will sit in the queue with no worker to pick it up.

The insidious part is that this is not an error condition from Temporal's perspective. The system is designed to handle workers being temporarily unavailable. So the task just waits. And waits. Your workflow is stuck, making no progress, and there are no error logs to alert you.

## Why?

This is one of the most common "silent failures" when working with Temporal. It manifests as workflows that appear to have started successfully (the `StartWorkflow` call returns a valid [workflow ID](terms/workflow-id.md) and run ID) but never make any progress.

What you will see is the Schedule-To-Start Latency (STSL) metric climbing. This metric measures the time between when a task is scheduled and when a worker picks it up. Under normal operation this should be very low (milliseconds). When no worker is listening on the task queue, STSL grows indefinitely.

Common causes include:
- Typos in task queue names (e.g. `"my-task-queue"` vs `"my_task_queue"`)
- A starter service and a worker service using different configuration values for the same logical task queue
- Deploying a new workflow type but forgetting to register the corresponding worker on the correct task queue
- Renaming a task queue in code but not updating all references

## How?

**Use shared constants or configuration for task queue names.** Never hardcode task queue strings in multiple places. Define them once and share them across your starter and worker code. In Go for example, define a package-level constant that both your starter and worker import.

**Monitor Schedule-To-Start Latency.** Set up alerts on STSL metrics. If STSL for any task queue exceeds a few seconds, something is likely wrong. This is your primary signal for task queue mismatches, missing workers, or worker capacity issues.

**Verify task queue names in your deployment pipeline.** If you use different services for starting workflows and running workers, add checks to validate that task queue names are consistent across services.

**Use the Temporal UI or CLI to inspect task queue state.** The `tctl` CLI or the Web UI can show you whether there are pollers (workers) registered on a given task queue. If you see tasks queued but zero pollers, you've found your problem.
