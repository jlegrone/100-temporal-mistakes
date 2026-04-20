# Doing Work Outside of the Workflow

> [!TIP]
> Work done before starting a workflow or sending a [signal](../terms/signals.md) is not durable -- if the process crashes between the side effect and the Temporal API call, state is lost.

A common pattern when first adopting Temporal is performing meaningful work in the caller process before starting a workflow:

<!--SNIPSTART doing-work-outside-bad-->
[doing_work_outside_of_the_workflow/examples.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/doing_work_outside_of_the_workflow/examples.go)
```go
// Dangerous: work done outside the workflow
func dangerousExample(ctx context.Context, temporalClient client.Client, db database, data Data, options client.StartWorkflowOptions) error {
	record, err := db.Insert(ctx, data)
	if err != nil {
		return err
	}
	// If the process crashes HERE, the record exists but no workflow was started
	_, err = temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, record.ID)
	return err
}

```
<!--SNIPEND-->

Temporal's durability only applies to code running inside a workflow or activity. The gap between completing external work and starting/signaling a workflow is a vulnerability window. This works 99.9% of the time -- the failures are rare but happen exactly when things are already going wrong (under load, during deployments).

Move the work inside the workflow instead:

<!--SNIPSTART doing-work-outside-good-->
[doing_work_outside_of_the_workflow/examples.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/doing_work_outside_of_the_workflow/examples.go)
```go
// Better: start the workflow first, let it do the work durably
func betterExample(ctx context.Context, temporalClient client.Client, data Data, options client.StartWorkflowOptions) error {
	_, err := temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, data)
	return err
}

func MyWorkflow(ctx workflow.Context, data Data) error {
	var record Record
	err := workflow.ExecuteActivity(ctx, InsertRecord, data).Get(ctx, &record)
	// If the worker crashes after insert, replay skips the completed activity
	_ = record
	return err
}

```
<!--SNIPEND-->

If you must do work before starting a workflow, make both the external work and the workflow start [idempotent](../terms/idempotency.md) so you can safely retry the entire operation.
