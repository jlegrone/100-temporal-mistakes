# Doing Work Outside of the Workflow

> [!TIP]
> * Work done before starting a workflow or before sending a [signal](terms/signals.md) is not durable -- if the process crashes, that work is lost.
> * Move side-effectful operations into workflows or activities to benefit from Temporal's durability guarantees.
> * The boundary between "your code" and "Temporal-managed code" is the most dangerous place for data loss.

## What?

A common pattern, especially when teams first adopt Temporal, is to perform meaningful work in the caller process before starting a workflow or sending a signal. For example:

```go
// Dangerous: work done outside the workflow
record, err := database.Insert(ctx, data)
if err != nil {
    return err
}

// If the process crashes here, the database record exists
// but no workflow was ever started to process it.
_, err = temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, record.ID)
```

The same problem applies to sending signals:

```go
result, err := externalAPI.Call(ctx, params)
if err != nil {
    return err
}

// If the process crashes here, the API call happened
// but the workflow never learns about the result.
err = temporalClient.SignalWorkflow(ctx, workflowID, runID, "channel", result)
```

## Why?

The core value of Temporal is durable execution -- your code makes progress even when processes crash, machines fail, or deployments happen. But this guarantee only applies to code that runs inside a workflow or an activity. Any work performed in a caller process (an API handler, a cron job, a CLI tool) is subject to the same failure modes as any ordinary program.

The gap between completing work and starting/signaling a workflow is a window of vulnerability. If the process dies in that gap, you end up with an inconsistent state: side effects have been applied to the external world but Temporal has no record of them and no way to recover.

This is particularly insidious because it works fine 99.9% of the time. The failures are rare but catastrophic, and they tend to happen exactly when things are already going wrong (under load, during deployments, during infrastructure issues).

## How?

Move the work inside the workflow:

```go
// Better: start the workflow first, let it do the work durably
_, err = temporalClient.ExecuteWorkflow(ctx, options, MyWorkflow, data)
```

```go
func MyWorkflow(ctx workflow.Context, data Data) error {
    var record Record
    // The database insert is now an activity -- if the worker crashes
    // after the insert, Temporal will replay and skip the completed activity.
    err := workflow.ExecuteActivity(ctx, InsertRecord, data).Get(ctx, &record)
    if err != nil {
        return err
    }
    // Continue processing...
}
```

If you truly need to coordinate between an external system and a workflow, consider these patterns:

1. **Start the workflow first**, then have the workflow perform side effects as activities. The workflow is the durable orchestrator.
2. **Use [idempotency](terms/idempotency.md) keys**: If you must do work before starting a workflow, make both the external work and the workflow start idempotent so you can safely retry the entire operation.
3. **Signal then act**: Start the workflow upfront (even eagerly), then send signals to it as events arrive. The workflow decides what to do with each signal durably.
