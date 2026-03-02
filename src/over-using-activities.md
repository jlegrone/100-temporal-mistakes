# Over-Using Activities

> [!TIP]
> * Every activity creates [history](terms/event-history.md) events, requires serialization of inputs and outputs, and adds a round-trip to the Temporal server.
> * Don't create an activity for every tiny operation -- batch related work into fewer, coarser-grained activities.
> * Workflows that schedule thousands of fine-grained activities risk [overflowing their history size](<overflowing-workflow-history-size.md>).

## What?

A common mistake is treating activities like function calls and creating one for every small operation. For example, instead of a single "process user" activity, developers create separate activities for "get user from database", "validate user fields", "format user response", and "log the result". Each of these adds overhead to the workflow.

This often comes from applying general software design principles (single responsibility, small functions) to Temporal without accounting for the cost model of activities.

## Why?

Activities are not free. Each activity execution generates at least three history events: `ActivityTaskScheduled`, `ActivityTaskStarted`, and `ActivityTaskCompleted` (or `ActivityTaskFailed`). These events are persisted in the Temporal server backend, transmitted over the network, and [replayed](terms/replay.md) every time the workflow rebuilds its state.

The costs add up:
- **History bloat**: A workflow that schedules hundreds of fine-grained activities accumulates thousands of history events, increasing replay time and pushing toward the [history size limit](<overflowing-workflow-history-size.md>).
- **Serialization overhead**: Every activity input and output is serialized and deserialized. If you pass a large object to five sequential activities that each transform it slightly, you are serializing that object ten times (five inputs, five outputs) instead of twice (one input, one output).
- **Network round-trips**: Each activity requires a round-trip between the [worker](terms/worker.md) and the Temporal server to schedule and report completion. Fine-grained activities turn a fast in-process operation into a networked one.
- **Latency**: Even if each round-trip is fast, chaining many sequential activities adds up. A sequence of 20 activities at 5ms overhead each adds 100ms of pure infrastructure latency.

## How?

**Group related operations into a single activity**: If multiple operations naturally belong together and don't individually need Temporal's retry or timeout semantics, combine them. Instead of three activities ("fetch", "validate", "transform"), write one activity ("process") that does all three.

**Use activities at the boundary**: Activities work best at the boundary between your workflow and external systems (databases, APIs, file systems). Internal logic like validation, transformation, or computation can happen inside the activity or in the workflow code itself (if it is deterministic and side-effect-free).

**Ask yourself**: "Does this operation need its own [retry policy](terms/retry-policy.md), timeout, or [heartbeat](terms/heartbeat.md)?" If the answer is no, it probably doesn't need to be a separate activity.

**Don't go too far the other way**: A single activity that runs for 30 minutes and does everything is also problematic -- it can't be partially retried and doesn't report progress. The goal is a sensible balance: coarse enough to avoid overhead, fine enough to enable meaningful retries and visibility.
