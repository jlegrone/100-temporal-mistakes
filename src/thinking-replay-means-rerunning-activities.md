# Thinking Replay Means Re-Running Activities

> [!TIP]
> During [replay](terms/replay.md), activities are not re-executed -- their results are loaded from history. Activities should still be idempotent, but because of retries and recovery, not because of replay.

A common misconception is that when a workflow replays, all of its activities run again. They do not. During replay, the workflow code re-executes from the beginning, but every time it encounters an activity call, the SDK checks the [history](terms/event-history.md) and returns the previously recorded result without calling the activity function. Replay is purely a local operation that reconstructs workflow state by matching commands against recorded events.

Misunderstanding this leads to unnecessary panic about side effects ("won't my database write happen twice?"), misplaced [idempotency](terms/idempotency.md) investment specifically to protect against replay, and over-engineering with guards or flags that try to "prevent" activities from running again -- guards which can actually introduce [non-determinism](terms/non-determinism.md) bugs if they cause the workflow to produce different commands on replay.

The correct reasons to make activities idempotent are retries (an activity can be retried after a transient failure or [worker](terms/worker.md) crash) and speculative execution (in some edge cases, Temporal may schedule an activity on multiple workers simultaneously). Both are runtime concerns unrelated to replay.
