# Event History

The event history is the immutable, append-only log of events that records everything that happened during a workflow execution. Every significant action -- activity scheduling, activity completion, timer creation, signal receipt, workflow completion -- is recorded as an event. The history is persisted in the Temporal server backend and is the source of truth for a workflow's state.

During replay, the event history is used to reconstruct the workflow's state by providing recorded results for activities, timers, and other operations rather than re-executing them. History has size limits: a default maximum of 50,000 events and a separate byte-size limit. When these limits are exceeded, the workflow is terminated.

## Related

- [Overflowing Workflow History Length](../overflowing-workflow-history-length.md)
- [Overflowing Workflow History Bytes](../overflowing-workflow-history-bytes.md)
- [Passing Too Much Information from Activities](../passing-too-much-information-from-activities.md)
- [Not Using Continue-As-New](../not-using-continue-as-new.md)
- [Downloading History with Decode Payloads](../downloading-history-with-decode-payloads.md)
- [Replay](replay.md)
- [Continue-As-New](continue-as-new.md)
- [Temporal Server Backend](temporal-server-backend.md)
- [Payload](payload.md)
