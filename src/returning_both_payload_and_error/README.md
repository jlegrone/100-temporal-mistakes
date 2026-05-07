# Returning Both a Payload and an Error

<!-- TODO: Add a unit test demonstrating this issue (payload being silently dropped when an error value is also returned). Also provide an example showing how to include and extract error details in a temporal application error. Putting structured details in the error itself is also more flexible than putting error details in the result because it allows the activity to choose whether the error is retryable or nonretryable depending on the temporal error type. But it's still valid to not return an error value at all if the intent is for the activity not to be retried. -->

<!-- TODO: Make the tip text less Go SDK specific. Also fact check that the Python and TypeScript SDKs have the same behavior. -->
> [!TIP]
> In Go, the Temporal SDK discards the [payload](../terms/payload.md) when an error is returned alongside it from an activity or workflow, leading to silent data loss.

In Go, returning both a non-nil value and a non-nil error is syntactically valid, and some developers use this pattern to return partial results alongside an error. However, the Temporal Go SDK records a failure event (not a completion event) when an error is returned, so **the payload is silently discarded** -- the caller receives the error but a zero-value result. This is particularly dangerous because it works fine when testing the activity function directly outside of Temporal, and no warning indicates the result was dropped.

Return either a result or an error, never both:

<!--SNIPSTART returning-both-payload-and-error-good-->
[returning_both_payload_and_error/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/returning_both_payload_and_error/activity.go)
```go

// Good: return either a result or an error, never both.
func MyActivity(ctx context.Context, input Input) (Result, error) {
	result, err := doWork(input)
	if err != nil {
		return Result{}, fmt.Errorf("failed to process: %w", err)
	}
	return Result{Data: result}, nil
}

```
<!--SNIPEND-->

<!-- TODO: Remove this approach (using return value rather than error value) in favor of using error details: https://pkg.go.dev/go.temporal.io/sdk@v1.42.0/internal#ApplicationErrorOptions. Make sure there is a unit test for the error details approach. Error details are more flexible because they allow you to leverage retry policy while ALSO returning structured data to the workflow when the retries are eventually exhausted. -->

<!-- TODO: Check if there is a way to read the error from the previous activity attempt from inside the current activity execution. This might enhance the example, but would at the least be good to mention. Something similar might be achieved with heartbeat details, but getting the last error would be preferable. -->

If you need to communicate partial results alongside a failure, encode the partial result into the result struct itself:

<!--SNIPSTART returning-both-payload-and-error-partial-->
[returning_both_payload_and_error/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/returning_both_payload_and_error/activity.go)
```go

// Good: encode partial results into the result struct itself.
func MyActivityPartial(ctx context.Context, input Input) (Result, error) {
	result, err := doWork(input)
	if err != nil {
		return Result{
			Data:       result,
			PartialErr: err.Error(),
		}, nil
	}
	return Result{Data: result}, nil
}

```
<!--SNIPEND-->
