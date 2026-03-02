# Not Using Workflow Versioning

> [!TIP]
> * Deploying changes to workflow code without versioning causes non-determinism errors for in-flight workflows.
> * Use the SDK's patching/versioning APIs (`workflow.GetVersion` in Go, `patched()` in TypeScript) to safely evolve workflow definitions.
> * All running workflows replay with the latest deployed code, so old and new code paths must coexist until old executions complete.

## What?

When you deploy a new version of your workflow code, all currently running workflows will [replay](terms/replay.md) using that new code. If the updated code produces different commands than what was originally recorded in the workflow's [history](terms/event-history.md), Temporal detects the mismatch and raises a [non-determinism](terms/non-determinism.md) error. The workflow gets stuck and stops making progress.

Temporal relies on deterministic replay to reconstruct workflow state. The workflow code re-executes from the beginning, and at each step Temporal checks that the commands produced match the events already in history. Changing the sequence, type, or parameters of those commands breaks that contract.

Common changes that trigger non-determinism errors include:
- Adding, removing, or reordering activity calls
- Changing timer durations
- Adding or removing [child workflow](terms/child-workflow.md) executions
- Modifying the arguments passed to an activity

## Why?

Without [versioning](terms/versioning.md), you can't safely evolve workflow logic while workflows are running. You'd have to drain all running workflows before every deployment, which is impractical for long-running workflows that may execute for days, weeks, or even months.

Non-determinism errors are particularly painful because they silently break workflows that were previously healthy. You deploy what looks like a harmless change, and suddenly hundreds of in-flight workflows start failing. Recovering from this often requires rolling back the deployment and then manually dealing with workflows that accumulated errors in the meantime.

## How?

Use the SDK's [versioning](terms/versioning.md) APIs to introduce changes safely. These APIs let you branch your workflow code so that existing executions continue following the old path while new executions take the new path.

In Go, use `workflow.GetVersion`:

```go
v := workflow.GetVersion(ctx, "change-id", workflow.DefaultVersion, 1)
if v == workflow.DefaultVersion {
    // Old code path: existing workflows execute this
    err = workflow.ExecuteActivity(ctx, OldActivity, input).Get(ctx, &result)
} else {
    // New code path: new workflows execute this
    err = workflow.ExecuteActivity(ctx, NewActivity, input).Get(ctx, &result)
}
```

In TypeScript, use `patched`:

```typescript
if (patched('change-id')) {
    // New code path
    result = await newActivity(input);
} else {
    // Old code path
    result = await oldActivity(input);
}
```

Keep the old code branch around until you are certain no running workflow will ever need to execute it. Only then can you safely remove it. Getting this lifecycle wrong is itself a common mistake -- see [Incorrect Workflow Patching](incorrect-workflow-patching.md) for details.

## Testing

Temporal SDKs provide replay testing utilities that let you take a workflow's history and replay it against your updated code. This is the most reliable way to verify that your versioning is correct before deploying. Feed recorded histories from production workflows into your test suite and assert that [replay](terms/replay.md) succeeds without errors.
