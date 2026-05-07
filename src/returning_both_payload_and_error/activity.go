package returning_both_payload_and_error

import (
	"context"
	"fmt"
)

// Input is the input type for the activity.
type Input struct{}

// SomeData represents the data produced by an activity.
type SomeData struct{}

// Result is the output type for the activity.
type Result struct {
	Data       SomeData
	PartialErr string // Empty if fully successful
}

func doWork(_ Input) (SomeData, error) {
	return SomeData{}, nil
}

// @@@SNIPSTART returning-both-payload-and-error-good

// Good: return either a result or an error, never both.
func MyActivity(ctx context.Context, input Input) (Result, error) {
	result, err := doWork(input)
	if err != nil {
		return Result{}, fmt.Errorf("failed to process: %w", err)
	}
	return Result{Data: result}, nil
}

// @@@SNIPEND

// @@@SNIPSTART returning-both-payload-and-error-partial

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

// @@@SNIPEND
