# Returning Both a Payload and an Error

> [!TIP]
> * In Go, the Temporal SDK discards the payload when an error is returned alongside it from an activity or workflow.
> * This leads to silent data loss -- the caller receives a nil result even though the activity produced meaningful output.
> * Return either a successful result or an error, never both.

## What?

In Go, functions commonly return `(result, error)` tuples. It is syntactically valid to return a non-nil value for both. Some developers use this pattern to return partial results alongside an error, for example:

```go
func MyActivity(ctx context.Context, input Input) (Result, error) {
    result, err := doWork(input)
    if err != nil {
        // Returning both the partial result and the error
        return result, fmt.Errorf("partial failure: %w", err)
    }
    return result, nil
}
```

In the Temporal Go SDK, when an activity or workflow function returns both a non-nil payload and a non-nil error, **the error takes precedence and the payload is discarded**. The caller on the other side gets the error but a zero-value result.

## Why?

The Temporal SDK serializes activity and workflow results into the event history. When an error is returned, the SDK records a failure event, not a completion event. There is no mechanism to store both a result payload and an error in the same event. The error wins, and the payload is silently dropped.

This is particularly dangerous because:

1. **Silent data loss**: The activity did produce a meaningful (possibly partial) result, but the caller never sees it. There is no warning that the result was discarded.
2. **Inconsistent behavior with plain Go**: In regular Go code, the caller can inspect both return values. Developers used to this pattern are surprised when Temporal drops the result.
3. **Misleading during development**: If you test your activity function directly (outside of Temporal), the partial result is available. The data loss only manifests when running through the Temporal SDK.

## Solution

Return either a result or an error, not both:

```go
func MyActivity(ctx context.Context, input Input) (Result, error) {
    result, err := doWork(input)
    if err != nil {
        // Option 1: Return only the error
        return Result{}, fmt.Errorf("failed to process: %w", err)
    }
    return result, nil
}
```

If you need to communicate partial results alongside a failure, encode the partial result into the error itself or use a result struct that carries both:

```go
// Option 2: Encode the outcome in the result struct
type Result struct {
    Data        SomeData
    PartialErr  string  // Empty if fully successful
}

func MyActivity(ctx context.Context, input Input) (Result, error) {
    result, err := doWork(input)
    if err != nil {
        // Return a "successful" result that describes the partial failure
        return Result{
            Data:       result,
            PartialErr: err.Error(),
        }, nil
    }
    return Result{Data: result}, nil
}
```

With this approach, the caller receives the full result (including the partial error description) because the activity completes successfully from Temporal's perspective. The caller can then inspect `Result.PartialErr` and decide how to handle the partial failure.

Note that this guidance is specific to the Go SDK. Other language SDKs may handle this differently based on their language's error handling conventions.
