# Not Using Workflow Replay for Debugging

> [!TIP]
> The Temporal SDK can replay a single workflow's [history](terms/event-history.md) locally, letting you step through the exact execution in a debugger. This is often the fastest way to reproduce and diagnose workflow bugs.

Temporal's [replay](terms/replay.md) mechanism is not just an internal runtime detail -- it is a debugging tool. Every Temporal SDK provides an API to replay a workflow execution from a history file. You point it at a downloaded history, and the SDK re-executes your workflow code step by step, exactly as it originally ran. This means you can set breakpoints in your IDE, inspect state at any point during execution, and reproduce production bugs locally without needing the original environment, database state, or timing conditions.

Download the workflow history using the Temporal CLI or Web UI, then replay it locally:

```bash
temporal workflow show \
  --workflow-id your-workflow-id \
  --run-id your-run-id \
  --output json > history.json
```

```go
func TestReplayWorkflow(t *testing.T) {
    replayer := worker.NewWorkflowReplayer()
    replayer.RegisterWorkflow(YourWorkflow)
    err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "history.json")
    require.NoError(t, err)
}
```

Set breakpoints in your workflow code and run the replay test under your debugger -- the workflow will execute exactly as it did in production. Once you have captured a history that exposed a bug, keep it as a test fixture. Replay tests make excellent regression tests because they verify that your current code can correctly process histories produced by previous versions, catching [non-determinism](terms/non-determinism.md) errors before deployment.

See also: [Not Validating Replay Safety Before Deployments](not-validating-replay-safety-before-deployments.md), [Downloading History with DecodePayloads Enabled](downloading-history-with-decode-payloads.md).
