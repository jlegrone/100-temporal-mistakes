# Part One: Activities

Need to be robust to:
- Downstream service errors
- Worker crashes & hangs
- Temporal server disruption

Should also:
- Not amplify bad requests
- Gracefully handle cancelation

---

Activities: Handling Downstream Service Errors

<!-- Screenshot of simple example activity that calls an generic payments API and returns the result (modeled after Stripe) -->



---

## Overview of Activity Timeouts & Retry Policy

<!-- 1. Screenshot of StartActivityOptions from the Go SDK -->
<!-- 2. Highlight the options in the screenshot that we'll talk about -->

---



---

## Activities are the workflow's connection to the outside world

The server owns the activity lifecycle.

- It can run more than once.
- It can be canceled.
- Its timeouts decide how long Temporal keeps trying.

---

## Mistake — Not making activities idempotent

Activities have **at-least-once** execution semantics.

Even with `MaximumAttempts = 1`:
- Workers crash after completing the work but before reporting back.
- Cluster failover can replay an attempt.
- The server schedules another attempt; you charge the card twice.

You cannot reliably reproduce this in dev.

---

## Pattern — Idempotent by construction

- Idempotency key derived from workflow ID + activity input.
- Pass it to downstream services that accept idempotency keys.
- Use database constraints: `INSERT ... ON CONFLICT DO NOTHING`.
- Prefer naturally idempotent operations: upsert, set-to-state.

Assume your activity will run more than once.

---

## Mistake — Preventing activity retries

Three quiet timeout misconfigurations that defeat retries:

| Misconfiguration | Result |
|---|---|
| No heartbeat timeout | Stuck worker undetected until start-to-close (or schedule-to-close) fires |
| No start-to-close timeout | One stuck attempt eats the whole retry budget |
| start-to-close == schedule-to-close | First attempt uses the entire budget; no time for retries |

---

## Pattern — Set all three timeouts

- **Heartbeat timeout** — fast detection of stuck workers.
- **Start-to-close timeout** — per-attempt deadline.
- **Schedule-to-close timeout** — total budget across retries.

Rule of thumb: `schedule-to-close ≥ start-to-close × max_attempts` plus margin for backoff.

---

## Mistake — Setting too-short timeouts

Common pattern: happy-path latency + a small margin.

A 30-second `schedule-to-close` against a 3-minute downstream outage:
- Activity retries a few times.
- Hits the schedule-to-close.
- Fails permanently — even though the next attempt would have succeeded.

Timeouts based on happy-path latency defeat the retry mechanism.

---

## Pattern — Base timeouts on outage tolerance

Ask: *"How long am I willing to wait for eventual success?"*

- `schedule-to-close` = outage tolerance.
- `start-to-close` = tighter, per-attempt deadline.
- `WorkflowExecutionTimeout` = even more margin across all activities.

---

## Mistake — Activity can't be canceled because it doesn't heartbeat

Activity cancellation is **cooperative**, delivered through heartbeat responses.

- Workflow requests cancellation.
- Server records it.
- Server delivers it on the next heartbeat response.

No heartbeats → activity runs to natural completion regardless.

---

## Pattern — Heartbeat from any cancelable activity

```go
for i, item := range input.Items {
    if ctx.Err() != nil {
        return ctx.Err()
    }
    processItem(item)
    activity.RecordHeartbeat(ctx, i)
}
```

- Set a `HeartbeatTimeout` on the activity options.
- Heartbeat at least every few seconds for activities running over 10 s.
- Carry progress in heartbeat details so a retry resumes instead of restarting from zero.

---

## Mistake — Calling external services without controlling retry behavior

Even with timeouts dialed in, retries can hurt you:

1. Retrying errors that aren't retryable.
2. Hammering a struggling downstream during an outage.

---

## Sub-mistake — Retrying non-retryable errors

Auth failures. Validation errors. "Not found."

Retrying them:
- Wastes the retry budget.
- Pollutes downstream logs and metrics.
- Delays surfacing the real failure to the workflow.

**Pattern**: decide retryability *from the activity* — return a non-retryable application error for known-terminal failures.

---

## Sub-mistake — Hammering a struggling downstream

Aggressive retries during resource exhaustion make the outage worse.

**Pattern**:
- Configure real backoff: `InitialInterval`, `BackoffCoefficient`, `MaximumInterval`.
- Honor downstream backpressure (e.g., `Retry-After`).
- Centralize policy in an interceptor so every team's activities behave the same way.

---

## The well-formed activity template

Every activity should be:

- **Idempotent** — at-least-once means at-least-once.
- **Fully timed out** — heartbeat + start-to-close + schedule-to-close.
- **Heartbeating** — with progress details for resumability.
- **Deliberate about retries** — non-retryable for terminal errors; backoff for the rest.

Now: workflows.
