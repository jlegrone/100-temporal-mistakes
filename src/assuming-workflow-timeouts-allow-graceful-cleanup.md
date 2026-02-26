# Assuming Workflow Timeouts Allow Graceful Cleanup

> [!TIP]
> * When a workflow execution timeout or run timeout is reached, Temporal [terminates](terms/terminate.md) the workflow -- it does not cancel it.
> * A terminated workflow gets no chance to run cleanup code: no compensation, no resource release, no notifications.
> * If you need graceful cleanup on timeout, implement a timer + cancellation pattern inside your workflow instead of relying on workflow-level timeouts.

## What?

A common assumption is that when a workflow times out, it receives some kind of cancellation signal and gets a chance to run cleanup logic -- compensation steps, releasing external resources, notifying downstream services. This is wrong.

Workflow execution timeouts and run timeouts are enforced by the Temporal server. When the timeout is reached, the server [terminates](terms/terminate.md) the workflow. Termination is the equivalent of `kill -9`: the workflow code simply stops running. No deferred functions execute, no cancellation handlers fire, no cleanup happens.

This catches people off guard because activity cancellation works differently. When you cancel a workflow via the API, activities receive a cancellation signal and can perform cleanup. But a timeout-triggered termination skips all of that.

## Why?

If your workflow relies on timeout-triggered cleanup for correctness -- say, releasing a distributed lock, sending a failure notification, or running compensation logic -- none of that code will ever execute when the timeout fires. You end up with leaked resources, inconsistent state, and no indication of what went wrong beyond a "Terminated" status in the UI.

This is particularly dangerous for workflows that hold external state. A workflow that acquires a lease on an external resource and expects to release it on timeout will simply leave that lease dangling forever.

## How?

Instead of relying on workflow-level timeouts for cleanup, implement the timeout yourself inside the workflow using a timer and cancellation scope:

1. **Use a workflow timer as your deadline.** Start a timer for your desired maximum duration. If the timer fires before the workflow completes, trigger cancellation of the in-progress work from within the workflow itself.

2. **Handle the cancellation.** In the cancellation handler, run your cleanup logic -- compensation, notifications, resource release. Because this is a cancellation (not a termination), your cleanup code gets a chance to execute.

3. **Keep the workflow-level timeout as a safety net.** Set the execution timeout to something longer than your internal timer (e.g., internal timer at 30 minutes, execution timeout at 1 hour). The execution timeout becomes a backstop for truly stuck workflows, not your primary timeout mechanism.

This pattern gives you the best of both worlds: graceful cleanup under normal timeout conditions, and a hard kill for cases where something has gone seriously wrong.
