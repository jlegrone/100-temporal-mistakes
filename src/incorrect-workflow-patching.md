# Incorrect Workflow Patching

> [!TIP]
> * Even when you know to use patching, doing it wrong causes the same non-determinism errors you were trying to avoid.
> * Never remove the old code branch until every running workflow has advanced past the patched point.
> * Test your patches with replay tests using real workflow histories before deploying.

## What?

[Versioning](terms/versioning.md) APIs like `workflow.GetVersion` and `patched()` are the right tool for evolving workflow code, but they have a lifecycle that must be respected. Getting the details wrong -- removing the old branch too early, using incorrect version numbers, or nesting patches improperly -- leads to the same non-determinism errors that patching was supposed to prevent.

Common mistakes include:

1. **Removing the old code branch prematurely.** You deploy version 2 of a patch, see it working, and clean up the old branch. But some long-running workflow started weeks ago and hasn't reached the patched code yet. When it does, it replays with the new code, finds no matching old branch, and fails.

2. **Wrong version numbers.** Incrementing version numbers incorrectly or reusing a change ID with different semantics breaks the mapping between history and code.

3. **Nesting patches incorrectly.** Placing a new patch inside an existing version branch can create combinations that don't replay correctly if the outer and inner patches interact in unexpected ways.

4. **Skipping the deprecation step.** The patching lifecycle typically has three phases: (a) add the patch with both old and new branches, (b) deprecate the patch and keep only the new branch with a deprecation marker, (c) remove the patch entirely. Jumping straight from (a) to (c) risks breaking workflows that already have the patch marker in their history.

## Why?

These mistakes are insidious because they pass all your regular tests. The new code works perfectly for new workflows. The problem only surfaces when an old workflow with existing history tries to [replay](terms/replay.md) through the patched section. This might not happen until days or weeks after deployment, making it hard to connect the failure to the change that caused it.

The blast radius scales with how many in-flight workflows exist. A single incorrect patch removal can break every workflow that was started before the change.

## How?

**Understand the patch lifecycle:**

1. **Introduce the patch.** Deploy code with both old and new branches. New workflows take the new branch; existing workflows take the old branch during replay.
2. **Wait.** Let all workflows that started before step 1 complete (or at least advance past the patched point). This is the step people skip.
3. **Deprecate the patch.** Remove the old branch but keep the version/patch marker so that workflows started during step 1 (which have the patch marker in their history) still replay correctly.
4. **Wait again.** Let all workflows started during steps 1-2 complete.
5. **Remove the patch entirely.** Now no running workflow has this patch marker in its history, so the marker can be safely removed.

**Use replay tests.** Record workflow histories from production and replay them against your updated code in CI. This catches non-determinism errors before they reach production.

**Be conservative with cleanup.** If you're unsure whether all affected workflows have completed, don't remove the old branch yet. The cost of keeping dead code around is low compared to the cost of breaking running workflows.

**Keep patches simple.** Avoid deeply nested patches. If a section of code needs frequent changes, consider restructuring it so the changing logic lives in an activity rather than in workflow code. Activity code can be changed freely without versioning concerns.
