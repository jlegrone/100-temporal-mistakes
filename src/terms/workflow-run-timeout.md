# Workflow Run Timeout

The workflow run timeout limits the duration of a single workflow run (a single execution in a continue-as-new chain, not the entire chain). If reached, the run is terminated. This is useful for ensuring that individual runs of a long-lived workflow complete within a bounded time, separate from the overall execution timeout.

If both a workflow run timeout and workflow execution timeout are set, the run timeout must be less than or equal to the execution timeout.

## Related

- [Not Setting a Workflow Timeout](../not-setting-a-workflow-timeout.md)
- [Assuming Workflow Timeouts Allow Graceful Cleanup](../assuming-workflow-timeouts-allow-graceful-cleanup.md)
- [Not Using ContinueAsNew](../not-using-continue-as-new.md)
- [Workflow Execution Timeout](workflow-execution-timeout.md)
- [Terminate](terminate.md)
- [ContinueAsNew](continue-as-new.md)
