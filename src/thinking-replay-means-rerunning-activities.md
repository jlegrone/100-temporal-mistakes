# Thinking Replay Means Re-Running Activities

> [!TIP]
> During [replay](terms/replay.md), no activities or side effects are re-executed -- instead their results are loaded from history.

A common misconception is that when a workflow replays, all of its activities run again. They do not. During replay, the workflow code re-executes from the beginning, but every time it encounters an activity call, the SDK checks the [history](terms/event-history.md) and returns the previously recorded result without calling the activity function. Replay is purely a local operation that reconstructs workflow state by matching commands against recorded events.

Note that it is still important to make activities idempotent due to retries (an activity can be retried after a transient failure or [worker](terms/worker.md) crash).

<!-- TODO: Link to activity idempotency mistake entry. -->
