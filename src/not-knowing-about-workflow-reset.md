# Not Knowing About Workflow Reset
<!-- TODO: Consider reframing around the mistake of "abandoning" workflows with failed (probably because of an incorrectly versioned workflow code change) workflow tasks (they can be rescued!) -->

> [!TIP]
> Workflow reset lets you rewind a workflow to a specific point in its history and continue forward with fixed code. It is invaluable for recovering from bugs without losing the work already completed.

Temporal records the complete history of every workflow execution. Workflow reset leverages this history to re-execute a workflow from a chosen point, discarding everything that happened after it. Events up to the reset point are preserved and [replayed](terms/replay.md); events after it are discarded; and the workflow continues executing from that point with whatever [worker](terms/worker.md) code is currently deployed.

Without knowing about reset, teams facing a workflow that went down the wrong path are left with bad options: [terminate](terms/terminate.md) and restart (losing all progress), or manual compensation (error-prone and unscalable). Reset gives you a targeted recovery mechanism: fix the bug, deploy the fix, and reset affected workflows to just before the faulty logic executed. Combined with [batch operations](not-knowing-about-batch-operations-api.md), you can reset millions of affected workflows in a few minutes.

To use reset, examine the [workflow history](terms/event-history.md) to find the event ID just before the workflow took the wrong path, then reset via the CLI:

```bash
temporal workflow reset \
  --workflow-id my-workflow \
  --run-id abc123 \
  --event-id 42 \
  --reason "Resetting to before bug in payment logic"
```

<!-- TODO: Add bad binary reset point example too -->

Deploy your fix first -- if you reset before fixing the code, the workflow will hit the same bug again. Activities that completed before the reset point will not re-execute (their recorded results are replayed), but activities after the reset point will, so make sure activity implementations are [idempotent](terms/idempotency.md). You can also use reset types like `LastWorkflowTask`, `LastContinuedAsNew`, or `BadBinary` to simplify targeting.
<!-- TODO: Fact check all types and verify semantics match the real API. -->

See also: [Not Knowing About the Batch Operations API](not-knowing-about-batch-operations-api.md).
<!-- Also link to activity idempotency mistake -->
