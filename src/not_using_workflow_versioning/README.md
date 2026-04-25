# Not Using Workflow Versioning

<!-- TODO: rename this to "patching" since that's what it's called in most SDKs. -->
<!-- TODO: revise this now that worker versioning is an option (but note that version is still necessary when using "unpinned" workflows). -->
<!-- TODO: add a note on how to safely remove handlers for old version numbers (leveraging search attributes). -- actually this should just go in the incorrect workflow patching entry, but link to it. -->
<!-- TODO: Copy framing text from my Replay 2022 presentation -->

> [!TIP]
> Deploying changes to workflow code without versioning causes replay errors for in-flight workflows. Use patching/versioning to safely evolve and maintain backwards compatibility of workflow code.

<!-- TODO: Fact check that changing timer durations is actually a non-replay-safe change. -->

When you deploy a new version of your workflow code, all currently running workflows will [replay](terms/replay.md) using that new code. If the updated code produces different commands than what was originally recorded in the workflow's [history](terms/event-history.md), Temporal detects the mismatch and raises a [non-determinism](terms/non-determinism.md) error, and the workflow gets stuck. Common changes that trigger this include adding, removing, or reordering activity calls, changing timer durations, modifying [child workflow](terms/child-workflow.md) executions, or changing activity arguments.

Non-determinism errors are particularly painful because they silently break workflows that were previously healthy -- you deploy what looks like a harmless change, and suddenly hundreds of in-flight workflows start failing.

Use the SDK's versioning APIs to introduce changes safely. In Go, use `workflow.GetVersion`; in TypeScript, use `patched()`. These APIs branch your workflow code so existing executions follow the old path while new executions take the new path:

<!-- TODO: update this example to use a switch statement instead of if/else. -->
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
