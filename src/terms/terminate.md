# Terminate
In Temporal, workflow termination is equivalent of a `kill -9` command in bash. It stops an ongoing workflow execution immediately without giving it a chance to perform any cleanup operation.

Unlike cancellation, which is cooperative and gives the workflow a chance to run cleanup code, termination is immediate and irrevocable. Terminated workflows record a `WorkflowExecutionTerminated` event as the final history event.

## Related

- [Terminating rather than canceling](../terminating-rather-than-canceling.md)
- [Assuming workflow timeouts allow graceful cleanup](../assuming-workflow-timeouts-allow-graceful-cleanup.md)
- [Not setting a workflow timeout](../not-setting-a-workflow-timeout.md)
- [Cancellation](cancellation.md)
- [Event History](event-history.md)
