# Terminating Rather Than Canceling

> [!TIP]
> [Termination](terms/terminate.md) is `kill -9` -- no cleanup runs. [Cancellation](terms/cancellation.md) is cooperative -- the workflow can run compensation logic before completing. Default to cancellation.

When operators need to stop a workflow, many default to termination because it feels decisive. But termination denies the workflow any opportunity to clean up: resources held (database locks, cloud infrastructure) aren't released, multi-step processes are left partially completed, and the workflow's final state tells you nothing about what was happening.

Cancellation is cooperative: the workflow receives a cancellation request, catches it, runs compensation logic, and completes gracefully. Default to `tctl workflow cancel` or the "Cancel" action in the UI. Reserve termination for true emergencies where the workflow is stuck in a tight loop or causing active harm. Consider restricting terminate permissions via [namespace](terms/namespace.md)-level access controls.

See also: [Not Using a Disconnected Context for Cleanup](not-using-disconnected-context-for-cleanup.md).
