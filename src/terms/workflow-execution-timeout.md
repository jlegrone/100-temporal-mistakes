# Workflow Execution Timeout

The workflow execution timeout limits the total time for a workflow execution, including all runs in a continue-as-new chain. When reached, the workflow is terminated (not canceled) -- no cleanup code runs. The default is effectively 10 years if not set.

This timeout acts as a safety net to prevent workflows from running indefinitely due to bugs. It should be set generously enough to accommodate the expected maximum duration of the workflow, including time for retries of failed activities and downstream service outages.

## Related

- [Not Setting a Workflow Timeout](../not-setting-a-workflow-timeout.md)
- [Assuming Workflow Timeouts Allow Graceful Cleanup](../assuming_workflow_timeouts_allow_graceful_cleanup/)
- [Workflow Run Timeout](workflow-run-timeout.md)
- [Terminate](terminate.md)
- [ContinueAsNew](continue-as-new.md)
