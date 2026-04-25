# Not Validating Replay Safety Before Deployments

<!-- TODO: Note that leveraging worker versioning https://docs.temporal.io/worker-versioning and exclusively using pinned workflows is an alternative strategy that avoids the need for constant checks. -->
<!-- TODO: Give advice on how to monitor for workflows that failed workflow tasks due to replay safety issues. -->
<!-- TODO: Nit: non-replay safe changes can themselves be deterministic. Nondeterminism is a different class of mistake than just making an unpatched/unversioned code change (which itself is determinsitically going down a different code path than the previous version of the workflow code). These two concepts are getting confused with each other in the text below. -->

> [!TIP]
> Replay tests against existing [workflow histories](terms/event-history.md) should be part of your CI/CD pipeline to catch non-replay safe changes before they reach production.

When you deploy new [worker](terms/worker.md) code, any running workflows will eventually [replay](terms/replay.md) using it. If the new code changes the workflow's deterministic sequence of commands -- adding, removing, or reordering activities without proper [versioning](terms/versioning.md) -- replay detects a mismatch and the workflow gets stuck with a [non-determinism](terms/non-determinism.md) error. Without automated replay validation in your deployment pipeline, you only discover these breaking changes after workflows start failing in production.

The fix is to run replay tests as part of CI/CD. Periodically export a representative sample of workflow histories from production, including histories from workflows in various states. Then write replay tests that feed these histories through your updated code:

<!--SNIPSTART not-validating-replay-safety-before-deployments-test-->
[not_validating_replay_safety_before_deployments/replay_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_validating_replay_safety_before_deployments/replay_test.go)
```go

func TestReplayWorkflowHistory(t *testing.T) {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(MyWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/my_workflow_history.json")
	require.NoError(t, err)
}

```
<!--SNIPEND-->

Automate the history collection with a scheduled job or CI step so your test fixtures stay current as workflows evolve. If replay tests fail, block the deployment -- the developer must either fix the non-deterministic change or add proper versioning guards. This approach catches problems deterministically before they affect production, rather than relying on manual review to spot unsafe changes.

See also: [Not Using Workflow Replay for Debugging](../not_using_workflow_replay_for_debugging/) for how replay testing can be used beyond deployment validation, [Not Using Workflow Versioning](../not_using_workflow_versioning/README.md), [Incorrect Workflow Patching](incorrect-workflow-patching.md).
