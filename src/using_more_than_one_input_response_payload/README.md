# Using More Than One Input/Response Payload

> [!TIP]
> Some SDKs (Go, Java) allow multiple positional arguments for workflow and activity inputs, but this breaks cross-SDK interoperability and makes schema evolution harder. Stick to a single struct/object for both input and output.

Some Temporal SDKs allow defining workflow and activity functions with multiple positional parameters, where each argument is serialized as a separate [payload](../terms/payload.md) in the [event history](../terms/event-history.md). This creates problems: the TypeScript and Python SDKs only support a single input argument, making cross-worker calls impossible. Adding, removing, or reordering positional arguments is also a breaking change; and call sites like `StartWorkflow("MyWorkflow", "user-123", 500, "USD")` make argument names opaque without reading the function signature.

Always wrap inputs into a single request object, and return a single result object rather than multiple values (beyond the error):

<!-- TODO: Add multiple (unnamed) return values (in addition to the error) in the v1 example as well. -->
<!-- TODO: Use "Request"/"Response" instead of "Input"/"Output" in code examples. -->
<!--SNIPSTART using-more-than-one-input-response-payload-workflow-->
[using_more_than_one_input_response_payload/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_more_than_one_input_response_payload/workflow.go)
```go

// Avoid this
func MyWorkflowV1(ctx workflow.Context, userID string, amount int, currency string) error {
	// ...
	_ = userID
	_ = amount
	_ = currency
	return nil
}

// Prefer this
type MyWorkflowInput struct {
	UserID   string
	Amount   int
	Currency string
}

func MyWorkflowV2(ctx workflow.Context, input MyWorkflowInput) error {
	// ...
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART using-more-than-one-input-response-payload-activity-->
[using_more_than_one_input_response_payload/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_more_than_one_input_response_payload/workflow.go)
```go

// Avoid this
func MyActivityV1(ctx context.Context, input MyActivityInput) (string, int, error) {
	// ...
	return "", 0, nil
}

// Prefer this
type MyActivityOutput struct {
	Status string
	Count  int
}

func MyActivityV2(ctx context.Context, input MyActivityInput) (MyActivityOutput, error) {
	// ...
	return MyActivityOutput{}, nil
}

```
<!--SNIPEND-->

This convention is a small upfront cost that pays for itself every time you evolve workflow contracts, interoperate across SDKs, or debug production issues.

See also: [Breaking changes to payloads](../breaking-changes-to-payloads.md).
