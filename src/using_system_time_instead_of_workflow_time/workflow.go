package using_system_time_instead_of_workflow_time

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART using-system-time-bad

// BAD: system time in workflow code
func MyWorkflowV1Time(ctx workflow.Context) error {
	now := time.Now() // Different on replay!
	if now.Hour() < 12 {
		// Morning logic
	}
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART using-system-time-good

// GOOD: workflow time
func MyWorkflowV2Time(ctx workflow.Context) error {
	now := workflow.Now(ctx) // Same value on replay
	if now.Hour() < 12 {
		// Morning logic -- deterministic
	}
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART using-system-time-sleep-bad

// BAD: language-native sleep
func MyWorkflowV1Sleep(ctx workflow.Context) error {
	time.Sleep(10 * time.Minute)
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART using-system-time-sleep-good

// GOOD: durable timer
func MyWorkflowV2Sleep(ctx workflow.Context) error {
	if err := workflow.Sleep(ctx, 10*time.Minute); err != nil {
		return err
	}
	return nil
}

// @@@SNIPEND
