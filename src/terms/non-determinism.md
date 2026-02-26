# Non-Determinism

Non-determinism in the Temporal context refers to workflow code that can produce different command sequences when re-executed with the same inputs and history. Since Temporal uses replay to reconstruct workflow state, non-deterministic code causes the replayed execution to diverge from the recorded history, resulting in a `NonDeterministicError` that blocks the workflow from making progress.

Common sources of non-determinism include: network calls, system time (`time.Now()` vs `workflow.Now()`), random number generation, environment variables, reading files, goroutines/threads not managed by the SDK, and modifying global/shared state.

## Related

- [Performing Network Calls in Workflow Code](../performing-network-calls-in-workflow-code.md)
- [Using System Time Instead of Workflow Time](../using-system-time-instead-of-workflow-time.md)
- [Reading Environment Variables in Workflow Code](../reading-environment-variables-in-workflow-code.md)
- [Modifying Shared State in Workflow Code](../modifying-shared-state-in-workflow-code.md)
- [Not Using Static Analysis / Sandboxed SDK](../not-using-static-analysis-sandboxed-sdk.md)
- [Not Using Workflow Versioning](../not-using-workflow-versioning.md)
- [Replay](replay.md)
- [Versioning](versioning.md)
- [Side Effect](side-effect.md)
- [Workflow Task](workflow-task.md)
