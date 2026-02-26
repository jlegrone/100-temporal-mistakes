# Thinking Replay Means Re-Running Activities

> [!TIP]
> * During [replay](terms/replay.md), activities are NOT re-executed -- their results are loaded from history.
> * Only workflow code is re-executed during replay; activity results are replayed from previously recorded events.
> * Activities should still be idempotent, but because of retries and recovery, not because of replay.

## What?

A very common misconception among Temporal newcomers is believing that when a workflow replays, all of its activities run again. This is not the case. During [replay](terms/replay.md), the workflow code is re-executed from the beginning, but every time it encounters a command to execute an activity, Temporal checks the history and finds the previously recorded result. The activity function itself is never called. The workflow simply receives the same result it got the first time and moves on.

This confusion often leads to reactions like: "If my workflow has 50 activities and it replays, won't it execute all 50 activities again? That sounds expensive and dangerous!" The answer is no -- replay is purely a local operation that reconstructs the workflow's state by matching commands against recorded history events.

## Why?

Misunderstanding replay leads to several downstream problems:

**Unnecessary panic about side effects.** Developers worry that a database write in an activity will happen twice during replay. It won't. The activity doesn't run during replay at all.

**Misplaced [idempotency](terms/idempotency.md) concerns.** Some teams invest heavily in making activities idempotent specifically to protect against replay. Activities absolutely should be idempotent, but the reason is retries (an activity can be retried after a transient failure or worker crash) and recovery, not replay. Conflating the two reasons leads to confused mental models.

**Over-engineering workflow code.** Developers sometimes add guards or flags in workflow code to "prevent" activities from running again during replay. These guards are unnecessary and can actually introduce [non-determinism](terms/non-determinism.md) bugs if they cause the workflow to produce different commands on replay.

## How?

Build the correct mental model of what happens during replay:

1. **The workflow function is called** just like it was the first time.
2. **When workflow code reaches an activity call**, the SDK checks the history. If a matching `ActivityTaskCompleted` (or `ActivityTaskFailed`) event exists, the SDK returns the recorded result without scheduling the activity.
3. **The workflow code continues** with that result, exactly as it did during the original execution.
4. **This repeats** until the workflow catches up to the point where history ends. From that point forward, new commands (activities, timers, etc.) are executed for real.

In practice, replay is what allows Temporal to recover workflow state after a [worker](terms/worker.md) crash or restart. The worker re-executes the workflow function, replays all the recorded results, and picks up exactly where it left off. No activity runs twice because of this process.

The correct reasons to make activities idempotent are:
- **Retries:** If an activity fails partway through (e.g., the worker crashes after writing to a database but before reporting success), Temporal will retry it on another worker.
- **Speculative execution:** In some edge cases, Temporal may schedule an activity on multiple workers simultaneously.

Both of these are runtime concerns unrelated to replay.
