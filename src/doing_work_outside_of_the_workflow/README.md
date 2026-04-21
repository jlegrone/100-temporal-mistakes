# Doing Work Outside of the Workflow

> [!TIP]
> Work done before starting a workflow or sending a [signal](../terms/signals.md) is not durable -- if the process crashes between the side effect and the Temporal API call, state is lost.

A common pattern when first adopting Temporal is performing meaningful work in the caller process before starting a workflow:

<!--SNIPSTART doing-work-outside-bad-->
[doing_work_outside_of_the_workflow/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/doing_work_outside_of_the_workflow/workflow.go)
```go

// HandleRequest performs a database insert before starting the workflow.
// If the process crashes between the insert and the workflow start, the
// record exists but no workflow is running to process it.
func (s *MyService) HandleRequest(ctx context.Context, data Data) error {
	record, err := s.DB.Insert(ctx, data)
	if err != nil {
		return err
	}
	// If the process crashes HERE, the record exists but no workflow was started.
	if _, err := s.Temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: "my-task-queue",
	}, MyWorkflowV1, record.ID); err != nil {
		return err
	}
	return nil
}

// MyWorkflowV1 receives the already-inserted record ID.
func MyWorkflowV1(ctx workflow.Context, id string) error {
	// ... process the record

	return nil
}

```
<!--SNIPEND-->

Temporal's durability only applies to code running inside a workflow or activity. The gap between completing external work and starting/signaling a workflow is a vulnerability window, where a system failure can leave behind dangling state.

Move the work inside the workflow instead:

<!--SNIPSTART doing-work-outside-good-->
[doing_work_outside_of_the_workflow/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/doing_work_outside_of_the_workflow/workflow.go)
```go

// HandleRequestV2 starts the workflow first and lets it perform the
// database insert as an activity, so both operations are durable.
func (s *MyService) HandleRequestV2(ctx context.Context, data Data) error {
	if _, err := s.Temporal.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: "my-task-queue",
	}, MyWorkflowV2, data); err != nil {
		return err
	}
	return nil
}

func MyWorkflowV2(ctx workflow.Context, data Data) error {
	var record Record
	if err := workflow.ExecuteActivity(ctx, InsertRecord, data).Get(ctx, &record); err != nil {
		return err
	}

	// ... continue processing with record.ID

	return nil
}

```
<!--SNIPEND-->

If you must do work before starting a workflow, make both the external work and the workflow start [idempotent](../terms/idempotency.md) so you can safely retry the entire operation.
