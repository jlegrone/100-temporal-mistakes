# Not Using ContinueAsNew

> [!TIP]
> Long-running workflows accumulate events in their history, leading to longer [replay](terms/replay.md) times and eventually hitting the [history length limit](overflowing-workflow-history-length.md). [ContinueAsNew](terms/continue-as-new.md) resets the history and also limits the age of your code, simplifying [versioning](terms/versioning.md).

Every action in a Temporal workflow -- scheduling an activity, receiving a [signal](terms/signals.md), firing a timer -- adds events to the [history](terms/event-history.md). For workflows that run indefinitely (event listeners, polling loops, subscription managers, recurring jobs), history grows without bound. Without [ContinueAsNew](terms/continue-as-new.md), the workflow eventually hits the 50k event limit and the server [terminates](terms/terminate.md) it. Even before that, large histories degrade performance: a [worker](terms/worker.md) replaying a [workflow task](terms/workflow-task.md) must process the entire history, and a workflow with 40,000 events takes significantly longer than one with 200. The longer a workflow runs, the more code versions it spans, requiring compatibility with code paths written months ago.

ContinueAsNew completes the current execution and immediately starts a new one with the same workflow ID, a fresh history, and whatever state you pass as the new input:

<!--SNIPSTART not-using-continue-as-new-workflow-->
[not_using_continue_as_new/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_continue_as_new/workflow.go)
```go

func SubscriptionWorkflow(ctx workflow.Context, state SubscriptionState) error {
	for {
		// ... do work ...

		// Check if it's time to continue as new
		if workflow.GetInfo(ctx).GetCurrentHistoryLength() > 10000 {
			return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
		}
	}
}

```
<!--SNIPEND-->

Trigger ContinueAsNew based on event count (e.g., 10,000 events), elapsed time (e.g., 24 hours, which caps code age and simplifies [versioning](terms/versioning.md)), or an explicit [signal](terms/signals.md) for operational control. When using it, carry over only the essential state the new execution needs -- this is your chance to compact state. Drain pending signal channels before continuing as new to avoid losing unprocessed signals (see [not draining signals before completing a workflow](../not_draining_signals_before_completing_workflow/)). Design the workflow input struct to serve as a checkpoint from the start, since it becomes the input to the new execution.

See also: [Overflowing workflow history length](overflowing-workflow-history-length.md), [Overflowing workflow history bytes](overflowing-workflow-history-bytes.md).
