# Non-Determinism

Non-determinism in the Temporal context refers to workflow code that can produce different command sequences when re-executed with the same inputs and history. Since Temporal uses replay to reconstruct workflow state, non-deterministic code causes the replayed execution to diverge from the recorded history, resulting in a `NonDeterministicError` that blocks the workflow from making progress.

Common sources of non-determinism include: network calls, system time (`time.Now()` vs `workflow.Now()`), random number generation, environment variables, reading files, goroutines/threads not managed by the SDK, and modifying global/shared state.

## Related

- [Performing Network Calls in Workflow Code](../performing_network_calls_in_workflow_code/README.md)
- [Using System Time Instead of Workflow Time](../using_system_time_instead_of_workflow_time/README.md)
- [Reading Environment Variables in Workflow Code](../reading_environment_variables_in_workflow_code/README.md)
- [Modifying Shared State in Workflow Code](../modifying_shared_state_in_workflow_code/README.md)
- [Not Using Static Analysis / Sandboxed SDK](../not-using-static-analysis-sandboxed-sdk.md)
- [Not Using Workflow Versioning](../not_using_workflow_versioning/README.md)
- [Replay](replay.md)
- [Versioning](versioning.md)
- [Side Effect](side-effect.md)
- [Workflow Task](workflow-task.md)
