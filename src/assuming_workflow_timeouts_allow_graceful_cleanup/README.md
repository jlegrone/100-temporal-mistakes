# Assuming Workflow Timeouts Allow Graceful Cleanup

> [!TIP]
> When a [workflow execution timeout](../terms/workflow-execution-timeout.md) fires, Temporal [terminates](../terms/terminate.md) the workflow -- it does not [cancel](../terms/cancelation.md) it. No cleanup code runs.

A common assumption is that a timed-out workflow receives a cancelation signal and gets a chance to run compensation logic, release resources, or send notifications. This is wrong. Timeout-triggered termination is the equivalent of `kill -9`: no deferred functions execute, no cancelation handlers fire. If your workflow holds external state (a distributed lock, a lease), it will be left dangling.

A workflow that relies on the execution timeout as its business deadline will be terminated without any opportunity to react:

<!--SNIPSTART assuming-workflow-timeouts-bad-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

// MyWorkflowV1 relies on the workflow execution timeout as its
// business deadline. It tries to compensate when the workflow is
// canceled, but when the timeout fires Temporal terminates the
// workflow -- the compensation never runs.
func MyWorkflowV1(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	err := workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow).Get(ctx, nil)
	if err != nil {
		// This code is unreachable on timeout: the workflow is terminated,
		// not canceled, so none of this executes.
		log.Warn("compensating", "error", err)
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})
		_ = workflow.ExecuteActivity(newCtx, CompensateActivity).Get(newCtx, nil)
		return err
	}
	return nil
}

```
<!--SNIPEND-->

If you need graceful behavior on timeout, implement the deadline yourself with a timer. If the timer fires before the workflow completes, the workflow can take action -- log, run compensation, or continue as new. Keep the workflow-level execution timeout as a safety net set to something longer (e.g., internal timer at 30 minutes, execution timeout at 1 hour).

<!--SNIPSTART assuming-workflow-timeouts-good-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow.go)
```go

func getSoftTimeout(ctx workflow.Context, padding time.Duration) (time.Duration, error) {
	info := workflow.GetInfo(ctx)
	// Prefer the run timeout if set; only fall back to execution timeout
	// if no run timeout is configured.
	timeout := info.WorkflowRunTimeout
	if timeout == 0 {
		timeout = info.WorkflowExecutionTimeout
	}
	if timeout > padding {
		return timeout - padding, nil
	}
	return 0, temporal.NewNonRetryableApplicationError(
		"Workflow timeout is too small",
		"wf_timeout_too_small",
		nil,
		map[string]string{
			"min_timeout": padding.String(),
		})
}

// MyWorkflowV2 uses an internal timer as the business deadline.
// If the timer fires before the child workflow completes, the
// workflow can react gracefully instead of being terminated.
func MyWorkflowV2(ctx workflow.Context) error {
	log := workflow.GetLogger(ctx)

	softTimeout, err := getSoftTimeout(ctx, time.Minute)
	if err != nil {
		return err
	}

	var (
		childFuture = workflow.ExecuteChildWorkflow(ctx, LongRunningWorkflow)
		selector    = workflow.NewSelector(ctx)
		selectError error
	)

	// Wait for child workflow
	selector.AddFuture(childFuture, func(f workflow.Future) {
		log.Info("child workflow completed")
	})
	// Wait for workflow cancelation
	selector.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		log.Warn("workflow canceled")
		selectError = ctx.Err()
	})
	// Wait for soft timeout
	selector.AddFuture(workflow.NewTimer(ctx, softTimeout), func(f workflow.Future) {
		log.Warn("deadline exceeded")
		selectError = workflow.ErrDeadlineExceeded
	})

	selector.Select(ctx)

	if selectError != nil {
		log.Warn("compensating", "error", selectError)
		newCtx, cancel := workflow.NewDisconnectedContext(ctx)
		defer cancel()
		newCtx = workflow.WithActivityOptions(newCtx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		})
		_ = workflow.ExecuteActivity(newCtx, CompensateActivity).Get(newCtx, nil)
		return selectError
	}

	return childFuture.Get(ctx, nil)
}

```
<!--SNIPEND-->

<!--SNIPSTART assuming-workflow-timeouts-test-->
[assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_workflow_timeouts_allow_graceful_cleanup/workflow_test.go)
```go

func TestMyWorkflowV2(t *testing.T) {
	type testCase struct {
		runTimeout    time.Duration
		setup         func(env *testsuite.TestWorkflowEnvironment)
		expectedError error
	}

	tests := map[string]testCase{
		"completes when child finishes": {
			// Timeout longer than the child's 1h sleep, so the child completes first.
			runTimeout: 2 * time.Hour,
		},
		"compensates on deadline": {
			// Timeout shorter than the child's 1h sleep, so the soft deadline fires first.
			runTimeout:    10 * time.Minute,
			expectedError: workflow.ErrDeadlineExceeded,
		},
		"compensates on cancelation": {
			runTimeout: 10 * time.Minute,
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterDelayedCallback(func() {
					env.CancelWorkflow()
				}, time.Second)
			},
			expectedError: &temporal.CanceledError{},
		},
		"fails on too-short timeout": {
			// Run timeout shorter than the padding (1m), so getSoftTimeout
			// returns an error before any work starts.
			runTimeout:    30 * time.Second,
			expectedError: &temporal.ApplicationError{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env := internaltestsuite.NewTestWorkflowEnvironment(t)
			env.RegisterWorkflow(LongRunningWorkflow)
			env.SetWorkflowRunTimeout(tc.runTimeout)

			compensated := false
			env.OnActivity(CompensateActivity, mock.Anything).Return(nil).Maybe().Run(
				func(args mock.Arguments) { compensated = true },
			)

			if tc.setup != nil {
				tc.setup(env)
			}

			env.ExecuteWorkflow(MyWorkflowV2)
			require.True(t, env.IsWorkflowCompleted())

			if tc.expectedError != nil {
				require.ErrorAs(t, env.GetWorkflowError(), &tc.expectedError)
			} else {
				require.NoError(t, env.GetWorkflowError())
			}
			// Compensation runs for operational errors (deadline, cancelation)
			// but not for configuration errors (too-short timeout).
			var appErr *temporal.ApplicationError
			wantCompensate := tc.expectedError != nil && !errors.As(tc.expectedError, &appErr)
			require.Equal(t, wantCompensate, compensated, "CompensateActivity called")
		})
	}
}

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_cancelled/) for more on handling cancelation in blocking workflows.
