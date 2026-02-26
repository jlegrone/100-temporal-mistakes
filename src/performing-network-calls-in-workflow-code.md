# Performing Network Calls in Workflow Code

> [!TIP]
> * Network calls (HTTP requests, database queries, gRPC calls) in workflow code break determinism and cause non-determinism errors on [replay](terms/replay.md).
> * Unlike activities, network calls made directly in workflow code will be re-executed on every replay, producing potentially different results each time.
> * Move all network I/O into activities, which are designed to handle non-deterministic operations safely.

## What?

Workflow code must be deterministic because it is re-executed during [replay](terms/replay.md) to reconstruct workflow state. Making network calls -- HTTP requests, database queries, gRPC calls, filesystem reads, or any other I/O -- directly in workflow code violates this requirement.

A network call is inherently [non-deterministic](terms/non-determinism.md): it might return different data, fail with a different error, have different latency, or time out entirely depending on when it runs. During replay, the call executes again (unlike activity calls, whose results come from history), and the different result causes the workflow to take a different code path, produce different commands, or fail outright.

```go
// BAD: network call in workflow code
func MyWorkflow(ctx workflow.Context) error {
    // This HTTP call will be re-executed on every replay
    resp, err := http.Get("https://api.example.com/config")
    if err != nil {
        return err
    }
    // The response may differ on replay, breaking determinism
    // ...
}
```

## Why?

The consequences of network calls in workflow code range from subtle to catastrophic:

**Non-determinism errors.** If the call returns different data on replay, the workflow produces different commands than what's in history. Temporal detects the mismatch and the workflow fails with a non-determinism error.

**Replay amplification.** Every time a workflow replays ([worker](terms/worker.md) restart, redeployment, cache eviction), the network call executes again. A workflow that replays 100 times makes 100 HTTP requests. This can overwhelm external services and add significant latency to replay.

**Silent data corruption.** If the network call happens to return the same shape of data but with different values (e.g., a config endpoint that was updated), the workflow may silently take a different path without triggering an obvious error.

**Unreliable error handling.** A network call that succeeded originally might fail on replay (or vice versa), causing unpredictable workflow behavior.

## How?

Move all network I/O into activities:

```go
// GOOD: network call in an activity
func FetchConfigActivity(ctx context.Context) (Config, error) {
    resp, err := http.Get("https://api.example.com/config")
    if err != nil {
        return Config{}, err
    }
    defer resp.Body.Close()
    var config Config
    err = json.NewDecoder(resp.Body).Decode(&config)
    return config, err
}

func MyWorkflow(ctx workflow.Context) error {
    var config Config
    // Activity result is recorded in history and replayed deterministically
    err := workflow.ExecuteActivity(ctx, FetchConfigActivity).Get(ctx, &config)
    if err != nil {
        return err
    }
    // Safe to use config here -- it's the same value on replay
    // ...
}
```

Activities are the correct abstraction for non-deterministic operations. Their results are recorded in [workflow history](terms/event-history.md) and returned directly during replay without re-executing the activity function.

If you need a small piece of non-deterministic data (like a random number or UUID) without the overhead of a full activity, use `workflow.SideEffect`. It executes the function once, records the result, and returns the recorded value on replay. However, for anything involving network I/O, prefer activities because they come with retries, timeouts, and [heartbeating](terms/heartbeat.md) built in.
