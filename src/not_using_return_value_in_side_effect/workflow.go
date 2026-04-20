package not_using_return_value_in_side_effect

import (
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART not-using-side-effect-return-bad

// WRONG: ignoring the return value
func MyWorkflowBad(ctx workflow.Context) error {
	var myUUID string
	workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		myUUID = uuid.New().String() // Sets variable as a side effect
		return nil
	})
	// During replay, the function doesn't run -- myUUID stays empty!
	_ = myUUID
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART not-using-side-effect-return-good

// CORRECT: using the returned value
func MyWorkflowGood(ctx workflow.Context) error {
	var myUUID string
	encodedValue := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return uuid.New().String()
	})
	if err := encodedValue.Get(&myUUID); err != nil {
		return err
	}
	// myUUID is correctly set during both first execution and replay
	_ = myUUID
	return nil
}

// @@@SNIPEND
