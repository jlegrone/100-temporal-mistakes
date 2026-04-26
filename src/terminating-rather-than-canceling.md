# Terminating Rather Than Canceling

<!-- TODO: Share temporal cli examples for cancel and batch cancel. -->

> [!TIP]
> Terminating workflows should only be done as a last resort. When you wish to stop a workflow execution, try cancelation instead. This allows the workflow to perform graceful cleanup and run compensating actions if it needs to.

See also: [Not Using a Disconnected Context for Cleanup](not_using_disconnected_context_for_cleanup/README.md).
<!-- TODO: Also link to deadlock on cancelation entry and resetting a stuck or terminated workflow entry. -->
