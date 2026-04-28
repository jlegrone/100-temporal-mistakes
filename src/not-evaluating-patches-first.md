# Not Evaluating Patches as the First Step in the Workflow

<!-- TODO: Write content. Key points to cover:
- `workflow.GetVersion` / `patched()` calls are what populate the `TemporalChangeVersion` search attribute.
- If the patch check isn't reached early in the workflow, you can't tell from search attributes / list filters whether a long-running workflow has executed past the patched point yet.
- This blocks the "wait for all workflows to advance past the patched point" step of the patching lifecycle (see incorrect-workflow-patching.md).
- Recommendation: place version checks at the top of the workflow function, before any long-running waits, signal loops, or activity calls that could block for extended periods.
- Show a before/after example with a workflow that calls GetVersion only inside a conditional branch deep in the workflow vs. at entry.
- Worst case: if the patch is evaluated inside a conditional branch that some workflows never enter, the `TemporalChangeVersion` search attribute will NEVER be set on those executions. A list-workflow query filtering by version will keep returning unversioned workflows indefinitely, making it impossible to automate the "is it safe to remove the old branch?" check.
-->

> [!TIP]
> TODO

See also: [Incorrect Workflow Patching](incorrect-workflow-patching.md), [Not Using Workflow Versioning](not_using_workflow_versioning/README.md).
