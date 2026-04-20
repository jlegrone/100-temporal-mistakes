# Modifying Workflow History or Behavior in Interceptors

> [!TIP]
> [Interceptors](terms/interceptor.md) run during [replay](terms/replay.md) just like workflow code. Changing an interceptor's behavior can cause [non-determinism](terms/non-determinism.md) errors for in-flight workflows -- treat interceptor code with the same [versioning](terms/versioning.md) discipline as workflow code.

Temporal SDK interceptors wrap workflow and activity execution for cross-cutting concerns like tracing, header injection, or policy enforcement. Because they participate in the command sequence recorded in [history](terms/event-history.md), changing an interceptor between the original execution and a replay causes the replayed commands to not match. A tracing interceptor that starts adding a new header, a shared library update that changes retry behavior, or adding/removing an interceptor entirely -- all break replay for in-flight workflows even though the workflow code itself hasn't changed.

Interceptors are dangerous precisely because they feel separate from workflow code. A team might carefully version their workflow definitions while freely updating a shared tracing interceptor. To avoid this: version interceptor changes the same way you version workflow changes, keep interceptors minimal (prefer read-only context propagation over execution flow modification), avoid feature flags in interceptors (the flag becomes part of the deterministic contract), and test interceptor changes with replay tests against existing workflow histories before deploying.

See also: [Not Validating Replay Safety Before Deployments](not-validating-replay-safety-before-deployments.md).
