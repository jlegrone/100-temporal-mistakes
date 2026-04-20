# Not Leveraging Workflow Input/Response Payloads

> [!TIP]
> Temporal supports typed inputs and outputs on workflow executions. Pass data as workflow arguments rather than feeding it via [signals](terms/signals.md) after the workflow starts.

Teams often start a workflow with no input and then send data through signals, turning the workflow into a state machine that waits for external events before it can do anything. This adds complexity for no benefit.

```go
// Avoid: signal-driven initialization
func OrderWorkflow(ctx workflow.Context) error {
    var orderID string
    ch := workflow.GetSignalChannel(ctx, "order-id")
    ch.Receive(ctx, &orderID)
    // ...
}

// Prefer: typed input and output
type OrderInput struct {
    OrderID string
}
type OrderOutput struct {
    Status      string
    CompletedAt time.Time
}

func OrderWorkflow(ctx workflow.Context, input OrderInput) (OrderOutput, error) {
    // Proceed directly with input.OrderID
}
```

Typed inputs make workflows self-documenting (the contract is clear from the function signature), easier to test (construct input, run, assert output), and better supported by tooling (the Temporal UI and CLI display typed [payloads](terms/payload.md)).
