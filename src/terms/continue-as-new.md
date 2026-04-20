# ContinueAsNew

ContinueAsNew completes the current workflow execution and immediately starts a new one with the same workflow ID but a fresh event history. The new execution carries over only the explicitly provided input -- all local state, pending timers, and history are discarded. This is the primary mechanism for keeping long-running workflows healthy by resetting history size and limiting code age, which simplifies versioning.

## Related

- [Not using ContinueAsNew](../not-using-continue-as-new.md)
- [Overflowing workflow history length](../overflowing-workflow-history-length.md)
- [Not draining signals before completing workflow](../not-draining-signals-before-completing-workflow.md)
- [Writing polling loops in workflow code](../writing-polling-loops-in-workflow-code.md)
- [Replay](replay.md)
- [Versioning](versioning.md)
- [Event History](event-history.md)
