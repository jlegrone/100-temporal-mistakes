# Assuming Workflow Timeouts Allow Graceful Cleanup

> [!TIP]
> When a [workflow execution timeout](terms/workflow-execution-timeout.md) fires, Temporal [terminates](terms/terminate.md) the workflow -- it does not [cancel](terms/cancellation.md) it. No cleanup code runs.

A common assumption is that a timed-out workflow receives a cancellation signal and gets a chance to run compensation logic, release resources, or send notifications. This is wrong. Timeout-triggered termination is the equivalent of `kill -9`: no deferred functions execute, no cancellation handlers fire. If your workflow holds external state (a distributed lock, a lease), it will be left dangling.

If you need graceful cleanup on timeout, implement it yourself: start a `workflow.NewTimer` for your desired maximum duration, and if it fires before the workflow completes, trigger cancellation of in-progress work from within the workflow. Your cancellation handler then runs cleanup logic. Keep the workflow-level execution timeout as a safety net set to something longer than your internal timer (e.g., internal timer at 30 minutes, execution timeout at 1 hour).

This gives you graceful cleanup under normal timeout conditions, and a hard kill for truly stuck workflows.
