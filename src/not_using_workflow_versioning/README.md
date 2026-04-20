# Not Using Workflow Versioning

> [!TIP]
> Deploying changes to workflow code without versioning causes non-determinism errors for in-flight workflows. Use the SDK's patching/versioning APIs to safely evolve workflow definitions while existing executions are still running.

When you deploy a new version of your workflow code, all currently running workflows will [replay](terms/replay.md) using that new code. If the updated code produces different commands than what was originally recorded in the workflow's [history](terms/event-history.md), Temporal detects the mismatch and raises a [non-determinism](terms/non-determinism.md) error, and the workflow gets stuck. Common changes that trigger this include adding, removing, or reordering activity calls, changing timer durations, modifying [child workflow](terms/child-workflow.md) executions, or changing activity arguments.

Without [versioning](terms/versioning.md), you cannot safely evolve workflow logic while workflows are running. You would have to drain all running workflows before every deployment, which is impractical for long-running workflows that may execute for days or months. Non-determinism errors are particularly painful because they silently break workflows that were previously healthy -- you deploy what looks like a harmless change, and suddenly hundreds of in-flight workflows start failing.

Use the SDK's versioning APIs to introduce changes safely. In Go, use `workflow.GetVersion`; in TypeScript, use `patched()`. These APIs branch your workflow code so existing executions follow the old path while new executions take the new path:

<!--SNIPSTART not-using-workflow-versioning-workflow-->
[not_using_workflow_versioning/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_workflow_versioning/workflow.go)
```go

func MyWorkflow(ctx workflow.Context, input Input) error {
	var result Result
	var err error

	v := workflow.GetVersion(ctx, "change-id", workflow.DefaultVersion, 1)
	if v == workflow.DefaultVersion {
		// Old code path: existing workflows execute this
		err = workflow.ExecuteActivity(ctx, OldActivity, input).Get(ctx, &result)
	} else {
		// New code path: new workflows execute this
		err = workflow.ExecuteActivity(ctx, NewActivity, input).Get(ctx, &result)
	}

	return err
}

```
<!--SNIPEND-->

Keep the old code branch around until you are certain no running workflow will ever need to execute it. Getting this lifecycle wrong is itself a common mistake -- see [Incorrect Workflow Patching](incorrect-workflow-patching.md) for details. Temporal SDKs also provide replay testing utilities that let you verify your versioning is correct before deploying; see [Not Validating Replay Safety Before Deployments](../not_validating_replay_safety_before_deployments/README.md).
