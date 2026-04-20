# Returning Both a Payload and an Error

> [!TIP]
> In Go, the Temporal SDK discards the [payload](../terms/payload.md) when an error is returned alongside it from an activity or workflow, leading to silent data loss.

In Go, returning both a non-nil value and a non-nil error is syntactically valid, and some developers use this pattern to return partial results alongside an error. However, the Temporal Go SDK records a failure event (not a completion event) when an error is returned, so **the payload is silently discarded** -- the caller receives the error but a zero-value result. This is particularly dangerous because it works fine when testing the activity function directly outside of Temporal, and no warning indicates the result was dropped.

Return either a result or an error, never both:

<!--SNIPSTART returning-both-payload-and-error-good-->
[returning_both_payload_and_error/example.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/returning_both_payload_and_error/example.go)
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

If you need to communicate partial results alongside a failure, encode the partial result into the result struct itself:

<!--SNIPSTART returning-both-payload-and-error-partial-->
[returning_both_payload_and_error/example.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/returning_both_payload_and_error/example.go)
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

With this approach, the activity completes successfully from Temporal's perspective, and the caller can inspect `Result.PartialErr` to decide how to handle the partial failure. This guidance is specific to the Go SDK; other language SDKs may handle this differently based on their error handling conventions.

See also: [Passing too much information from activities](../passing_too_much_information_from_activities/).
