# ContinueAsNew

ContinueAsNew completes the current workflow execution and immediately starts a new one with the same workflow ID but a fresh event history. The new execution carries over only the explicitly provided input -- all local state, pending timers, and history are discarded. This is the primary mechanism for keeping long-running workflows healthy by resetting history size and limiting code age, which simplifies versioning.

## Related

- [Not using ContinueAsNew](../not_using_continue_as_new/README.md)
- [Overflowing workflow history length](../overflowing-workflow-history-length.md)
- [Not draining signals before completing workflow](../not_draining_signals_before_completing_workflow/)
- [Writing polling loops in workflow code](../writing_polling_loops_in_workflow_code/README.md)
- [Replay](replay.md)
- [Versioning](versioning.md)
- [Event History](event-history.md)
