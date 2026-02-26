# Local Activity

A local activity is executed within the same workflow task as the workflow code, without scheduling a separate task through the server. This reduces latency (no round-trip to the server) and history size (local activities create fewer events). However, local activities must complete within the workflow task timeout (default 10 seconds), and their retries happen within the same task, counting against that timeout.

Local activities are appropriate for short, fast, reliable operations like reading from a local cache or calling a low-latency service. They are not suitable for operations that may take a long time, fail frequently, or need independent retry budgets.

## Related

- [Using Local Activities](../using-local-activities.md)
- [Fallible Local Activities](../fallible-local-activities.md)
- [Exceeding 10s Task Timeout](../exceeding-10s-task-timeout.md)
- [Activity Task](activity-task.md)
- [Workflow Task](workflow-task.md)
