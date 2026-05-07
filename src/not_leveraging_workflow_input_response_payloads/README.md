# Not Leveraging Workflow Input/Response Payloads

<!-- TODO: Reframe this around other workflow engines like airflow or dbt or databricks, because people coming from those may not be used to the pattern of request/response style control flow as opposed to storing intermediate artifacts in shared storage between tasks. -->

> [!TIP]
> Temporal supports typed inputs and outputs on workflow executions. Pass data as workflow arguments rather than feeding it via [signals](terms/signals.md) after the workflow starts.

Teams often start a workflow with no input and then send data through signals, turning the workflow into a state machine that waits for external events before it can do anything. This adds complexity for no benefit.

<!--SNIPSTART not-leveraging-payloads-bad-->
[not_leveraging_workflow_input_response_payloads/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_leveraging_workflow_input_response_payloads/workflow.go)
```go

// Avoid: signal-driven initialization
func OrderWorkflowBad(ctx workflow.Context) error {
	var orderID string
	ch := workflow.GetSignalChannel(ctx, "order-id")
	ch.Receive(ctx, &orderID)
	_ = orderID
	// ...
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART not-leveraging-payloads-good-->
[not_leveraging_workflow_input_response_payloads/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_leveraging_workflow_input_response_payloads/workflow.go)
```go

// Prefer: typed input and output
type OrderInput struct {
	OrderID string
}
type OrderOutput struct {
	Status      string
	CompletedAt time.Time
}

func OrderWorkflowGood(ctx workflow.Context, input OrderInput) (OrderOutput, error) {
	// Proceed directly with input.OrderID
	return OrderOutput{
		Status:      "completed",
		CompletedAt: workflow.Now(ctx),
	}, nil
}

```
<!--SNIPEND-->

Typed inputs make workflows self-documenting (the contract is clear from the function signature), easier to test (construct input, run, assert output), and better supported by tooling (the Temporal UI and CLI display typed [payloads](terms/payload.md)).
