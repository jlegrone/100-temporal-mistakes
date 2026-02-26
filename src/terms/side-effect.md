# Side Effect

SideEffect is a Temporal SDK primitive that allows running a small non-deterministic function within workflow code. The function is executed once during the initial execution and its return value is recorded in the event history. On subsequent replays, the recorded value is returned without re-executing the function.

SideEffect is intended for simple non-deterministic values like generating a UUID or reading a config value. It should not be used for operations with external side effects (like sending an email) -- those belong in activities. The critical rule is to always use the return value of SideEffect, as ignoring it means the non-deterministic function runs during replay, defeating the purpose.

MutableSideEffect is a variant that can be updated over time -- it re-executes on each replay and compares the new result with the previously recorded value, only recording a new event if the value has changed.

## Related

- [Not Using Return Value in Side Effect](../not-using-return-value-in-side-effect.md)
- [Reading Environment Variables in Workflow Code](../reading-environment-variables-in-workflow-code.md)
- [Performing Network Calls in Workflow Code](../performing-network-calls-in-workflow-code.md)
- [Replay](replay.md)
- [Non-Determinism](non-determinism.md)
- [Event History](event-history.md)
