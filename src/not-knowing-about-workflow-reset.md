# Not Knowing About Workflow Reset

> [!TIP]
> * Workflow reset allows you to [replay](terms/replay.md) a workflow from a specific point in its history, effectively "rewinding" it.
> * This is invaluable for recovering from bugs -- fix the code, then reset the workflow to before the bad decision was made.
> * Many teams don't know this feature exists and resort to manual workarounds when workflows go down the wrong path.

## What?

Temporal records the complete history of every workflow execution. Workflow reset leverages this history to re-execute a workflow from a chosen point, discarding everything that happened after that point. The workflow picks up from the reset point and continues forward using the current (presumably fixed) code.

Concretely, when you reset a workflow to event N in its history:
- Events up to N are preserved and replayed.
- Events after N are discarded.
- The workflow continues executing from that point with whatever [worker](terms/worker.md) code is currently deployed.

This is conceptually similar to `git reset` -- you're rewinding to a known good state and replaying from there.

## Why?

Without knowing about reset, teams facing a workflow that went down the wrong path due to a bug are left with bad options:

- **[Terminate](terms/terminate.md) and restart**: Loses all progress. If the workflow had completed expensive operations (payments, external API calls, provisioning), you may not be able to simply redo them.
- **Manual compensation**: Writing ad-hoc scripts or manually fixing state is error-prone and doesn't scale.
- **Waiting it out**: Hoping the workflow will eventually reach a recovery path, which may never happen if the bug is in a critical decision point.

Reset gives you a targeted recovery mechanism. You fix the bug in your workflow code, deploy the fix, and reset the affected workflows to the point just before the faulty logic executed. The workflows then proceed correctly using the new code.

This is particularly powerful when combined with [batch operations](not-knowing-about-batch-operations-api.md) -- if a bug affected hundreds of workflows, you can reset them all at once.

## How?

1. **Identify the reset point**. Examine the [workflow history](terms/event-history.md) in the Temporal UI or via `tctl` to find the event ID just before the workflow took the wrong path. Typically you'll want to reset to a [workflow task](terms/workflow-task.md) completed event.

2. **Reset via CLI or API**:
   ```bash
   # Reset a single workflow to a specific event ID
   tctl workflow reset --workflow-id my-workflow --run-id abc123 --event-id 42 --reason "Resetting to before bug in payment logic"

   # Reset to the last workflow task before a failure
   tctl workflow reset --workflow-id my-workflow --run-id abc123 --reset-type LastWorkflowTask --reason "Recovering from bug fix deployed in v2.3.1"
   ```

3. **Deploy your fix first**. Reset replays history and then continues with current code. If you reset before deploying the fix, the workflow will just hit the same bug again.

4. **Understand the implications**: activities that already completed before the reset point will not re-execute -- their recorded results are replayed. Activities after the reset point will be re-executed. Make sure activity implementations are [idempotent](terms/idempotency.md) or that re-execution is acceptable.

5. **Use reset types** to simplify targeting: `LastWorkflowTask`, `LastContinuedAsNew`, `BadBinary`, or a specific `EventId`. The `BadBinary` option is especially useful when you know which binary version introduced the bug.
