# Using More Than One Input/Response Payload

> [!TIP]
> * Some SDKs (Go, Java) allow multiple positional arguments as workflow and activity inputs -- this is a trap.
> * Other SDKs (TypeScript, Python) only support a single input argument, making cross-SDK interoperability painful when multiple arguments are used.
> * Stick to a single struct/object as workflow and activity input and output for consistency, portability, and easier evolution.

## What?

Some Temporal SDKs, notably Go and Java, allow defining workflow and activity functions with multiple positional parameters. For example in Go:

```go
func MyWorkflow(ctx workflow.Context, userID string, amount int, currency string) error {
    // ...
}
```

While this compiles and works, it creates several problems. Each positional argument is serialized as a separate [payload](terms/payload.md) in the Temporal [event history](terms/event-history.md). The workflow contract is implicitly defined by the order and types of positional arguments rather than by an explicit named structure.

## Why?

**Cross-SDK portability**: The TypeScript and Python SDKs only support a single input argument for workflows and activities. If you call a Go workflow from a TypeScript client (or vice versa), multiple positional arguments become a serialization headache. A single struct serializes to a single JSON object that any SDK can deserialize.

**Schema evolution**: Adding, removing, or reordering positional arguments is a breaking change. With a single struct, you can add new optional fields without breaking existing callers. You can also deprecate fields gradually.

**Readability**: `StartWorkflow("MyWorkflow", "user-123", 500, "USD")` is unclear without looking at the function signature. With a struct, `StartWorkflow("MyWorkflow", OrderInput{UserID: "user-123", Amount: 500, Currency: "USD"})` is self-documenting.

**Testing**: Constructing and asserting on a single input struct is cleaner than juggling multiple arguments.

## How?

Always wrap your workflow and activity inputs into a single struct:

```go
// Avoid this
func MyWorkflow(ctx workflow.Context, userID string, amount int, currency string) error {
    // ...
}

// Prefer this
type MyWorkflowInput struct {
    UserID   string
    Amount   int
    Currency string
}

func MyWorkflow(ctx workflow.Context, input MyWorkflowInput) error {
    // ...
}
```

The same applies to activity functions. And for outputs, return a single result struct rather than multiple return values (beyond the error):

```go
// Avoid this
func MyActivity(ctx context.Context, input MyActivityInput) (string, int, error) {
    // ...
}

// Prefer this
type MyActivityOutput struct {
    Status string
    Count  int
}

func MyActivity(ctx context.Context, input MyActivityInput) (MyActivityOutput, error) {
    // ...
}
```

This convention is a small upfront cost that pays for itself every time you need to evolve your workflow contracts, interoperate across SDKs, or debug production issues.
