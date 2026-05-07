# Modifying Workflow History or Behavior in Interceptors

> [!TIP]
> [Interceptors](terms/interceptor.md) run during [replay](terms/replay.md) just like workflow code. Changing an interceptor's behavior can cause [non-determinism](terms/non-determinism.md) errors for in-flight workflows -- treat interceptor code with the same [versioning](terms/versioning.md) discipline as workflow code.

Temporal SDK interceptors wrap workflow and activity execution for cross-cutting concerns like tracing or policy enforcement. Because they participate in the command sequence recorded in [history](terms/event-history.md), emitting workflow tasks from an interceptor causes the replayed commands to not match when the interceptor is added, removed, or updated.

See also: [Not Validating Replay Safety Before Deployments](not_validating_replay_safety_before_deployments/README.md).

<!-- TODO: If you use an interceptor to execute an activity or do anything else that modifies workflow history, you must ensure that changes to the interceptor are strictly aligned with your worker deployment, or that you are onboarded to worker versioning and exclusively use pinned workflows. -->
