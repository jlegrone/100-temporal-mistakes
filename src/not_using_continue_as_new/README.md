# Not Using ContinueAsNew

<!-- TODO: include a suggestion to implement time-based continue as new (in order to limit the max age of workflow executions and improve efficience of https://docs.temporal.io/worker-versioning with pinned workflows). -->

<!-- TODO: Note that it's ok to continue as new somewhat frequently, eg. between every state change. Confirm that it's not more expensive than starting a new child workflow. -->

<!-- TODO: include an example "ShouldContinueAsNew helper that also accounts for time. Double check the temporal issue tracker and source code to see if there is already a feature request or implementation of time based continue as new first though.

// ShouldContinueAsNew returns true if the workflow should continue as new soon.
// This value may change throughout the life of the workflow. Note that this is
// only a recommendation; workflows are allowed to continue as new at any time.
//
// ContinueAsNew may be suggested for one of several reasons:
//  1. The workflow history length is nearing limit
//  2. The workflow history total bytes is nearing limit
//  3. The workflow execution has been running for more than 24 hours
//  4. The workflow received a signal on the channel named `should_continue_as_new`
//
// Why 24 hours? This is a bit arbitrary, but the idea is to avoid forcing anyone into
// refactoring workflows to support ContinueAsNew if executions last for less than a day.
// When workflows can run for longer than this, reasoning about how soon old code paths will
// be phased out from running executions starts becoming more difficult. 24 hours seems like
// a reasonable point where the complexity of supporting ContinueAsNew is worth it.
func ShouldContinueAsNew(ctx Context) bool {
	// XXX(jlegrone): Changes to any behavior of this function (eg. modifying continueAsNewMaxDuration)
	//                must be gated by a version check. Skipping a version check would cause replay
	//                errors for running workflows.

	// Return true if workflow history size is growing large
	if GetInfo(ctx).GetContinueAsNewSuggested() {
		return true
	}

	// Return true if workflow has been running for longer than max duration
	continueAsNewMaxDuration := 24 * time.Hour
	if timeSinceWorkflowStart(ctx) > continueAsNewMaxDuration {
		return true
	}

	// Return true if workflow has received a ContinueAsNew signal
	for _, name := range workflow.GetUnhandledSignalNames(ctx) {
		if name == "should_continue_as_new" {
			return true
		}
	}

	return false
}

 -->

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

		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
		}
	}
}

```
<!--SNIPEND-->

Trigger ContinueAsNew based on event count (e.g., 10,000 events), elapsed time (e.g., 24 hours, which caps code age and simplifies [versioning](terms/versioning.md)), or an explicit [signal](terms/signals.md) for operational control. When using it, carry over only the essential state the new execution needs -- this is your chance to compact state. Drain pending signal channels before continuing as new to avoid losing unprocessed signals (see [not draining signals before completing a workflow](../not_draining_signals_before_completing_workflow/)). Design the workflow input struct to serve as a checkpoint from the start, since it becomes the input to the new execution.

See also: [Overflowing workflow history length](overflowing-workflow-history-length.md), [Overflowing workflow history bytes](overflowing-workflow-history-bytes.md).
