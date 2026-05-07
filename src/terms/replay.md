# Replay

Replay is the mechanism by which Temporal reconstructs a workflow's state. When a worker picks up a workflow task, it re-executes the workflow code from the beginning, using recorded results from the event history instead of re-executing activities, timers, and other operations. This is how Temporal achieves durability without persisting the entire workflow memory state.

Because workflow code is re-executed during replay, it must be deterministic -- producing the same sequence of commands given the same history. Non-deterministic code (network calls, system time, random values, etc.) will cause replay to diverge from the recorded history, resulting in a non-determinism error.

## Related

- [Thinking replay means rerunning activities](../thinking-replay-means-rerunning-activities.md)
- [Not using workflow replay for debugging](../not_using_workflow_replay_for_debugging/)
- [Not using workflow versioning](../not_using_workflow_versioning/README.md)
- [Performing network calls in workflow code](../performing_network_calls_in_workflow_code/README.md)
- [Using system time instead of workflow time](../using_system_time_instead_of_workflow_time/README.md)
- [Event History](event-history.md)
- [Non-determinism](non-determinism.md)
- [Versioning](versioning.md)
- [Workflow Task](workflow-task.md)
