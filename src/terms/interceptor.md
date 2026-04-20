# Interceptor

An interceptor is middleware that wraps workflow and activity execution, allowing cross-cutting concerns like logging, metrics, tracing, authentication, and header propagation to be implemented without modifying individual workflow or activity code. Interceptors are registered on the worker and run for every workflow task and activity task.

Because interceptors execute during both normal execution and replay, they are part of the deterministic contract. Changes to interceptor behavior (adding/removing interceptors, modifying how they wrap commands) can cause non-determinism errors for in-flight workflows. Interceptor code changes must be versioned with the same care as workflow code.

## Related

- [Modifying Workflow History in Interceptors](../modifying-workflow-history-in-interceptors.md)
- [Not Using Temporal SDK for Observability](../not_using_temporal_sdk_for_observability/)
- [Replay](replay.md)
- [Non-determinism](non-determinism.md)
- [Versioning](versioning.md)
- [Worker](worker.md)
