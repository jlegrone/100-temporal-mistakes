# Overflowing Workflow History Size

> [!TIP]
> The Temporal server [terminates](terms/terminate.md) workflows that exceed the maximum history size (50k events by default) with no chance for cleanup. Use [ContinueAsNew](terms/continue-as-new.md) to reset the history before it grows too large.

Temporal workflows have a hard limit on history size. When a workflow crosses the 50k event limit (adjustable through [dynamic configuration](terms/dynamic-config.md)), the server [terminates](terms/terminate.md) it with no chance for cleanup. Even before hitting the hard limit, large histories cause performance problems: [replay](terms/replay.md) must process the entire history to reconstruct state, and a 10ms replay delay compounds fast across millions of workflows.

Design workflows that accumulate events to use [ContinueAsNew](terms/continue-as-new.md). Trigger it when the event count reaches a threshold (e.g., 10,000 events), the workflow has been running longer than a reasonable limit (e.g., 24 hours), or a [signal](terms/signals.md) requests it explicitly. Capping workflow age through time-based triggers also simplifies [versioning](terms/versioning.md) and enables the [Temporal Worker Kubernetes Controller](terms/temporal-worker-kubernetes-controller.md) to manage fewer concurrent versions.

See also: [Not using ContinueAsNew](not-using-continue-as-new.md), [Overflowing workflow history bytes](overflowing-workflow-history-bytes.md).
