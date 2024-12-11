# Overflowing Workflow History Size

> [!TIP]
> * Workflows are terminated by default when they reach the maximum history size limit (50k events by default).
> * Large histories take longer to replay, potentially impacting performance when dealing with millions of workflows.
> * Solution for Long-Running Workflows: Use [ContinueAsNew](terms/continue-as-new.md) to trigger a new workflow instance when certain conditions are met, such as event count, time limits, or a signal reception.

## What?
Temporal workflows have hard limits. Default behavior when they are reached is for the server to [terminate](terms/terminate.md) the workflow meaning you don't have a chance for cleanup.

One hard limit is workflow maximum history size. By default, workflows are hard terminated when they cross the 50k events in their history limit. The limit can be tweaked via servers [dynamic configuration](<terms/dynamic-config.md>) values.

## Why?

[Replay](terms/replay.md) is not a free operation, especially when in memory caches are not hot. Large histories take more time to replay than short ones. 10ms additional replay delay may compound quickly when you run millions of workflows and you suddenly have to replay all of them.

## How?

By default, workflows which are known to accumulate events should be designed with [ContinueAsNew](terms/continue-as-new.md) in mind. You must be able to

Think about triggering [ContinueAsNew](terms/continue-as-new.md) when one of those things:
- Events counts
- Time is greater than reasonable limit e.g. 24h.
- A signal is received specifically to trigger ContinueAsNew

The timing aspect is important to ensure long running workflows age is capped which greatly simplifies [versioning](terms/versioning.md) and enables [temporal worker kubernetes controller](<terms/temporal-worker-kubernetes-controller.md>) to manage less versions.
