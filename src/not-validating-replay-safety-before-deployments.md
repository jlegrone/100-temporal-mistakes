# Not Validating Replay Safety Before Deployments

> [!TIP]
> Replay tests against production [workflow histories](terms/event-history.md) should be part of your CI/CD pipeline to catch non-deterministic changes before they reach production.

When you deploy new [worker](terms/worker.md) code, any running workflows will eventually [replay](terms/replay.md) using it. If the new code changes the workflow's deterministic sequence of commands -- adding, removing, or reordering activities without proper [versioning](terms/versioning.md) -- replay detects a mismatch and the workflow gets stuck with a [non-determinism](terms/non-determinism.md) error. Without automated replay validation in your deployment pipeline, you only discover these breaking changes after workflows start failing in production.

The fix is to run replay tests as part of CI/CD. Periodically export a representative sample of workflow histories from production, including histories from workflows in various states. Then write replay tests that feed these histories through your updated code:

```go
func TestReplayWorkflowHistory(t *testing.T) {
    replayer := worker.NewWorkflowReplayer()
    replayer.RegisterWorkflow(MyWorkflow)
    err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/my_workflow_history.json")
    require.NoError(t, err)
}
```

Automate the history collection with a scheduled job or CI step so your test fixtures stay current as workflows evolve. If replay tests fail, block the deployment -- the developer must either fix the non-deterministic change or add proper versioning guards. This approach catches problems deterministically before they affect production, rather than relying on manual review to spot unsafe changes.

See also: [Not Using Workflow Replay for Debugging](not-using-workflow-replay-for-debugging.md) for how replay testing can be used beyond deployment validation, [Not Using Workflow Versioning](not-using-workflow-versioning.md), [Incorrect Workflow Patching](incorrect-workflow-patching.md).
