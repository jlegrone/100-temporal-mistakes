# Not Validating Replay Safety Before Deployments

> [!TIP]
> * Deploying non-replay-safe workflow code causes running workflows to fail with [non-determinism](terms/non-determinism.md) errors.
> * Replay tests against production [workflow histories](terms/event-history.md) should be part of your CI/CD pipeline to catch breaking changes before they reach production.
> * Combine replay testing with proper [versioning](terms/versioning.md) to safely evolve workflow code over time.

## What?

When you deploy new [worker](terms/worker.md) code, any running workflows will eventually [replay](terms/replay.md) using it. If the new code changes the workflow's deterministic sequence of commands (e.g., added, removed, or reordered activities without proper [versioning](terms/versioning.md)), replay detects a mismatch between the recorded history and the expected commands. This produces a non-determinism error, and the workflow gets stuck.

Without automated replay validation in your deployment pipeline, you only discover these breaking changes after deployment, when workflows start failing in production.

## Why?

Temporal workflows are deterministic state machines. The history records every command the workflow issued (schedule activity, start timer, etc.), and during replay, the SDK re-executes the workflow code and verifies it produces the same sequence. Any deviation is a non-determinism error.

Common changes that break replay safety:
- Adding or removing an activity call without a version guard
- Changing the order of activities or [child workflow](terms/child-workflow.md) calls
- Modifying timer durations for in-flight timers
- Changing the type or structure of data stored in workflow state
- Switching from `activity.Execute` to `workflow.SideEffect` or vice versa

These changes are fine for **new** workflows, but **existing** workflows that have already recorded events in their history will break. The risk scales with the number of running workflows: if you have thousands of long-running workflows, a non-determinism bug is catastrophic.

## How?

**Run replay tests as part of CI/CD.** The core idea is simple: take real workflow histories from production and replay them using the new code. If replay succeeds, the code is safe to deploy.

1. **Collect production histories.** Periodically export a representative sample of workflow histories from production. Include histories from workflows in various states (running, completed, failed) to maximize coverage.

2. **Write replay tests.** Most Temporal SDKs provide a replay testing utility. In Go:

```go
func TestReplayWorkflowHistory(t *testing.T) {
    replayer := worker.NewWorkflowReplayer()
    replayer.RegisterWorkflow(MyWorkflow)

    // Replay against a production history file
    err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/my_workflow_history.json")
    require.NoError(t, err)
}
```

3. **Automate history collection.** Set up a scheduled job or CI step that fetches recent workflow histories and updates your test fixtures. This ensures your replay tests stay current as workflows evolve.

4. **Block deployments on replay failure.** If replay tests fail, halt the deployment. The developer must either fix the non-deterministic change or add proper [versioning](terms/versioning.md) guards.

See also: [Not using workflow replay for debugging](not-using-workflow-replay-for-debugging.md) for more on how replay testing can be used beyond deployment validation.
