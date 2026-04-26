# Starting Workflows from Activities

<!-- TODO: Use the dev server to demonstrate top level workflow cancelation not propagating to a workflow started by an activity. -->
<!-- TODO: Note that in situations where you are starting workflows from an activity, it might be better to use a child workflow with parent workflow close policy of ABANDON instead (that way you still see the parent-child relationship in the temporal workflow history/temporal UI for debugging). -->

> [!TIP]
> Starting a workflow from an activity hides the relationship from [history](../terms/event-history.md), breaks [cancellation](../terms/cancelation.md) propagation, and risks duplicate workflows on retry. Use [child workflows](../terms/child-workflow.md) from workflow code instead, even if the child workflow's, lifecycle must be decoupled from the parent.

When you start a workflow from an activity using the SDK client, the parent-child relationship is invisible in [event history](../terms/event-history.md), cancellation doesn't propagate, the parent can't await the child's result, and if the activity retries (e.g. after a timeout), you may end up with duplicate workflows unless you carefully set a deterministic [workflow ID](../terms/workflow-id.md).

<!--SNIPSTART starting-workflows-from-activities-good-->
[starting_workflows_from_activities/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/starting_workflows_from_activities/workflow.go)
```go
// GOOD: child workflow from workflow code
func MyWorkflowV2(ctx workflow.Context, input MyInput) (MyResult, error) {
	childFuture := workflow.ExecuteChildWorkflow(ctx, MyChildWorkflow, input)
	var result MyResult
	err := childFuture.Get(ctx, &result)
	return result, err
}

```
<!--SNIPEND-->

<!--SNIPSTART starting-workflows-from-activities-bad-->
[starting_workflows_from_activities/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/starting_workflows_from_activities/workflow.go)
```go
// BAD: starting a workflow from an activity
func MyActivity(ctx context.Context, input MyInput) error {
	c, _ := client.Dial(client.Options{})
	_, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{}, SomeWorkflow, input)
	return err
}

```
<!--SNIPEND-->

Using the SDK client to start workflows from outside a [worker](../terms/worker.md) (e.g. an HTTP handler) is perfectly fine. The anti-pattern is specifically about using the client inside an activity when a child workflow would be more appropriate.

<!-- TODO: add an explanation for how to start disconnected child workflows decoupled from the parent workflow lifecycle. This is one of the most common reasons for starting workflows from activities. -->