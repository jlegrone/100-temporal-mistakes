# Modifying Workflow History or Behavior in Interceptors

> [!TIP]
> * [Interceptors](terms/interceptor.md) run during workflow execution *and* during [replay](terms/replay.md) -- changes to interceptor behavior affect running workflows the same way changes to workflow code do.
> * Updating a shared library or interceptor that modifies how commands are issued (adding/removing activities, changing headers, wrapping calls) can cause [non-determinism](terms/non-determinism.md) errors for in-flight workflows.
> * Treat interceptor code with the same [versioning](terms/versioning.md) discipline as workflow code.

## What?

Temporal SDKs support interceptors (sometimes called middleware) that wrap workflow and activity execution. Interceptors are commonly used for:

- Adding tracing or logging context.
- Injecting headers into activity and [child workflow](terms/child-workflow.md) calls.
- Enforcing policies (timeouts, retries).
- Modifying inputs or outputs.

Because interceptors run as part of the workflow execution pipeline, they participate in the command sequence recorded in history. If an interceptor's behavior changes between the original execution and a [replay](terms/replay.md), the replayed command sequence won't match the recorded [history](terms/event-history.md), resulting in a non-determinism error.

The same risk applies to shared libraries that interceptors (or workflow code) depend on. A seemingly innocent library update can change behavior in ways that break replay.

## Why?

Interceptors are dangerous here precisely because they feel separate from workflow code. A team might carefully version their workflow definitions while freely updating a shared tracing interceptor or a utility library, not realizing that the interceptor is part of the deterministic contract.

Common scenarios that cause problems:

- **A tracing interceptor starts adding a new header** to activity calls. Old histories don't have that header, so replay produces a different command than what was recorded.
- **A shared library update changes retry behavior** inside an interceptor. Activities that were retried with one policy during the original execution are now retried differently during replay.
- **An interceptor is added or removed** between the original execution and replay. The command sequence changes because the interceptor was adding or modifying commands.
- **An interceptor conditionally modifies behavior** based on configuration or feature flags that change over time.

In all these cases, the workflow code hasn't changed -- but the effective behavior has, and that's what matters for determinism.

## How?

1. **Version interceptor changes like workflow changes.** If an interceptor change alters the command sequence (adding/removing activities, changing how calls are wrapped), use [versioning](terms/versioning.md) to ensure old workflows continue with the old behavior.

2. **Be cautious with shared library updates.** Before updating a library used by interceptors or workflow code, check whether the update changes any behavior that participates in the command sequence. Review changelogs and test with [replay](terms/replay.md) against existing workflow histories.

3. **Keep interceptors minimal.** The less an interceptor does, the less likely it is to break determinism. Prefer interceptors that only read data (e.g., propagating context) over ones that modify execution flow.

4. **Test interceptor changes with replay tests.** Download histories from running workflows and replay them with the new interceptor code before deploying. This catches non-determinism issues before they affect production. See [using workflow replay for debugging](not-using-workflow-replay-for-debugging.md) for how to set this up.

5. **Avoid feature flags in interceptors.** If an interceptor's behavior depends on a runtime flag or configuration value, that flag becomes part of the deterministic contract. Changing the flag breaks replay for workflows that ran with the old value.
