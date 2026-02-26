# Terminating Rather Than Canceling

> [!TIP]
> * [Termination](terms/terminate.md) is immediate and gives the workflow no chance to clean up -- it is the equivalent of `kill -9`.
> * Cancellation is cooperative: the workflow receives a cancellation request and can run cleanup logic before completing.
> * Default to cancellation unless you have a specific reason to terminate immediately.

## What?

When operators need to stop a running workflow, Temporal offers two mechanisms: **terminate** and **cancel**. Many teams default to termination because it feels decisive and they see it first in the UI or CLI, but in doing so they deny the workflow any opportunity to perform cleanup.

Termination ends the workflow immediately. No more workflow code runs. Any resources held, side effects in progress, or child workflows are left in an undefined state.

Cancellation, on the other hand, is cooperative. Temporal delivers a cancellation request to the workflow which can then catch it, run compensation logic (rolling back transactions, releasing external resources, notifying downstream systems), and complete gracefully.

## Why?

Reaching for terminate when cancel would suffice is dangerous for several reasons:

- **No cleanup**: Activities that allocated external resources (database locks, cloud infrastructure, third-party reservations) won't get a chance to release them. You are left with leaked resources that require manual intervention.
- **Inconsistent state**: If the workflow was coordinating a multi-step process, termination leaves it partially completed with no record of what was or wasn't cleaned up.
- **Lost observability**: A terminated workflow's final state gives you almost no information about what was happening at the time. A cancelled workflow can log its cleanup steps and complete with a meaningful result or error.

Termination has its place -- for example when a workflow is stuck in a tight loop due to a bug and will never process a cancellation request, or when you need to forcibly stop a workflow that is causing harm right now. But these cases are the exception, not the rule.

## How?

1. **Default to cancellation** in your operational runbooks and tooling. When you need to stop a workflow, use `tctl workflow cancel` or the "Cancel" action in the Temporal UI rather than "Terminate".

2. **Design workflows to handle cancellation** by wrapping cleanup logic in a deferred cancellation scope. In Go, this means using `workflow.NewDisconnectedCtx` and handling the `CanceledError`. In TypeScript, use `CancellationScope` with cleanup handlers.

3. **Reserve termination for true emergencies** where the workflow cannot or should not be allowed to run any more code at all. Document in your runbooks when termination is appropriate versus cancellation.

4. **Consider restricting terminate permissions** via namespace-level access controls so that only administrators can terminate workflows while regular operators can only cancel them.
