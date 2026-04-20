# Using System Time Instead of Workflow Time

> [!TIP]
> `time.Now()` returns different values on [replay](terms/replay.md), breaking determinism. Use `workflow.Now()` for the current time and `workflow.Sleep()` for delays -- both are deterministic and durable.

Using the system clock directly in workflow code is a common determinism violation. `time.Now()` in Go (or `Date.now()` in TypeScript, `datetime.now()` in Python) returns the current wall-clock time, which differs every time the code executes. Since workflow code re-executes during replay, this causes the workflow to potentially make different decisions.

<!--SNIPSTART using-system-time-bad-->
[using_system_time_instead_of_workflow_time/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_system_time_instead_of_workflow_time/workflow.go)
```go

// BAD: system time in workflow code
func MyWorkflowBadTime(ctx workflow.Context) error {
	now := time.Now() // Different on replay!
	if now.Hour() < 12 {
		// Morning logic
	}
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART using-system-time-good-->
[using_system_time_instead_of_workflow_time/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_system_time_instead_of_workflow_time/workflow.go)
```go

// GOOD: workflow time
func MyWorkflowGoodTime(ctx workflow.Context) error {
	now := workflow.Now(ctx) // Same value on replay
	if now.Hour() < 12 {
		// Morning logic -- deterministic
	}
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART using-system-time-sleep-bad-->
[using_system_time_instead_of_workflow_time/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_system_time_instead_of_workflow_time/workflow.go)
```go

// BAD: language-native sleep
func MyWorkflowBadSleep(ctx workflow.Context) error {
	time.Sleep(10 * time.Minute)
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART using-system-time-sleep-good-->
[using_system_time_instead_of_workflow_time/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/using_system_time_instead_of_workflow_time/workflow.go)
```go

// GOOD: durable timer
func MyWorkflowGoodSleep(ctx workflow.Context) error {
	if err := workflow.Sleep(ctx, 10*time.Minute); err != nil {
		return err
	}
	return nil
}

```
<!--SNIPEND-->

System time violations are especially tricky because they often work fine in development -- replay happens so quickly that `time.Now()` returns a nearly identical value. The bug only surfaces when a workflow replays hours or days later after a long [worker](terms/worker.md) outage or redeployment.

Temporal timers (`workflow.Sleep`, `workflow.NewTimer`) are both deterministic and durable: they survive worker restarts and produce the same behavior on replay. In the TypeScript SDK, `Date.now()` and `setTimeout` are automatically patched inside workflow code, but importing external libraries that use raw system time internally can still break determinism.
