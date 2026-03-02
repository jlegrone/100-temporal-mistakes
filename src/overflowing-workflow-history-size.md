# Overflowing Workflow History Size

> [!TIP]
> * The Temporal server [terminates](terms/terminate.md) workflows that exceed the maximum history size (50k events by default) -- no cleanup runs.
> * Large histories slow down [replay](terms/replay.md), which compounds across millions of workflows.
> * Use [ContinueAsNew](terms/continue-as-new.md) to reset the history before it grows too large.

## What?

Temporal workflows have hard limits on history size. When a workflow crosses the 50k event limit, the server [terminates](terms/terminate.md) it with no chance for cleanup. You can adjust this limit through [dynamic configuration](terms/dynamic-config.md), but the fundamental constraint remains: histories cannot grow without bound.

## Why?

[Replay](terms/replay.md) is not free, especially when in-memory caches are cold. Large histories take longer to replay than short ones. A 10ms replay delay compounds fast when you run millions of workflows and must replay all of them at once.

## How?

Design workflows that accumulate events to use [ContinueAsNew](terms/continue-as-new.md). Trigger it when:
- The event count reaches a threshold (e.g. 10,000 events)
- The workflow has been running longer than a reasonable limit (e.g. 24 hours)
- A [signal](terms/signals.md) requests it explicitly

Capping workflow age through time-based triggers also simplifies [versioning](terms/versioning.md) and enables the [Temporal Worker Kubernetes Controller](terms/temporal-worker-kubernetes-controller.md) to manage fewer concurrent versions.
