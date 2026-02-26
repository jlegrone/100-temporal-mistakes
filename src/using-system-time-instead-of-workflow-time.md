# Using System Time Instead of Workflow Time

> [!TIP]
> * Calling `time.Now()`, `Date.now()`, or equivalent in workflow code returns different values on [replay](terms/replay.md), breaking determinism.
> * Temporal SDKs provide deterministic time APIs (`workflow.Now()` in Go, `Date.now()` is automatically overridden in TypeScript) that return consistent values during replay.
> * When you need to wait for a duration or until a specific time, use Temporal timers (`workflow.Sleep`, `workflow.NewTimer`) instead of language-native sleep.

## What?

Using the system clock directly in workflow code is a common determinism violation. Calls like `time.Now()` in Go, `Date.now()` in TypeScript/JavaScript, or `datetime.now()` in Python return the current wall-clock time, which is different every time the code executes. Since workflow code is re-executed during [replay](terms/replay.md), the time value will differ from the original execution, potentially causing the workflow to make different decisions and produce different commands.

```go
// BAD: system time in workflow code
func MyWorkflow(ctx workflow.Context) error {
    now := time.Now() // Returns a different value on replay!
    if now.Hour() < 12 {
        // Morning logic
    } else {
        // Afternoon logic
    }
    // ...
}
```

This workflow might take the morning branch during original execution and the afternoon branch during replay, causing a [non-determinism](terms/non-determinism.md) error.

## Why?

System time violations are especially tricky because they often work fine in development and testing. The replay typically happens so quickly after the original execution that `time.Now()` returns a nearly identical value, and the workflow makes the same decision. The bug only surfaces when a workflow replays hours or days later -- after a long [worker](terms/worker.md) outage, a redeployment, or workflow cache eviction -- and the time difference causes a different code path.

This makes the issue hard to reproduce and diagnose. The workflow worked for weeks, and then one day it fails with a non-determinism error that seems to come out of nowhere.

## How?

Use the SDK's deterministic time API. During original execution, it returns the current time. During replay, it returns the time that was recorded in the workflow's [history](terms/event-history.md), guaranteeing the same value both times.

```go
// GOOD: workflow time
func MyWorkflow(ctx workflow.Context) error {
    now := workflow.Now(ctx) // Returns the same value on replay
    if now.Hour() < 12 {
        // Morning logic -- deterministic
    } else {
        // Afternoon logic -- deterministic
    }
    // ...
}
```

For delays and scheduling, use Temporal timers instead of language-native sleep:

```go
// BAD
time.Sleep(10 * time.Minute)

// GOOD
workflow.Sleep(ctx, 10 * time.Minute)
```

Temporal timers are durable (they survive worker restarts) and deterministic (they produce the same behavior on replay). A `workflow.Sleep` of 10 minutes creates a timer event in history. During replay, the SDK sees the timer event and skips past it instantly rather than waiting again.

Note that in the TypeScript SDK, `Date.now()` and `setTimeout` are automatically patched inside workflow code to be deterministic. However, it is still worth understanding why this matters, because importing an external library that uses raw system time internally can still break determinism.
