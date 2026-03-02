# Not Leveraging Workflow Input/Response Payloads

> [!TIP]
> * Temporal supports typed inputs and outputs on workflow executions -- use them.
> * Defining clear input and output types makes workflows self-documenting, easier to test, and enables better tooling.
> * Prefer passing data as workflow arguments rather than sending it via [signals](terms/signals.md) or [queries](terms/queries.md) after the workflow starts.

## What?

Not all workflow engines support typed inputs and outputs on workflow executions. Temporal does, and this powerful feature is often underused.

Teams new to Temporal often start a workflow with no input (or minimal input) and then feed it data through signals or queries after it starts. This turns the workflow into a state machine that waits for external events before it can do anything useful, adding complexity for no benefit.

Instead, workflows should accept well-defined typed input structs and return well-defined typed output structs. This makes the workflow's contract explicit and clear from the function signature alone.

## Why?

Leveraging workflow input and output [payloads](terms/payload.md) provides several benefits:

**Self-documenting workflows**: When a workflow's input type contains all the data it needs, anyone reading the code immediately understands what the workflow expects. No need to trace signal or query handlers to understand the data flow.

**Easier testing**: Testing a workflow that takes a typed input and returns a typed output is straightforward -- construct the input, run the workflow, assert on the output. Testing a workflow that relies on signals arriving in a specific order is significantly more complex.

**Better tooling**: The Temporal UI, CLI (`temporal workflow start`), and observability tools display and work with typed inputs and outputs. When everything the workflow needs is in its input payload, debugging and inspecting running workflows becomes much easier.

**Type safety**: Typed inputs catch errors at compile time (in typed languages) or at deserialization time rather than at runtime deep in your workflow logic.

## How?

Define a struct (or class/object depending on your SDK) that contains all the data your workflow needs to begin execution, and another for the result.

For example, instead of starting a workflow with no input and signaling it with an order ID:

```go
// Avoid this pattern
func OrderWorkflow(ctx workflow.Context) error {
    var orderID string
    ch := workflow.GetSignalChannel(ctx, "order-id")
    ch.Receive(ctx, &orderID)
    // now proceed with the order
}
```

Pass the order ID as part of the workflow input:

```go
// Prefer this pattern
type OrderInput struct {
    OrderID string
}

type OrderOutput struct {
    Status      string
    CompletedAt time.Time
}

func OrderWorkflow(ctx workflow.Context, input OrderInput) (OrderOutput, error) {
    // proceed directly with input.OrderID
}
```

The workflow caller can pass the input directly, the workflow can start processing immediately, and the result is returned as a typed value that the caller can inspect.
