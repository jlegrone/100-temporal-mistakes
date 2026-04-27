# Part One: Activities

Need to be robust to:
- Downstream service errors
- Worker crashes & hangs
- Temporal server disruption

Should also:
- Not amplify bad requests
- Gracefully handle cancelation

---

## Activities: Handling Downstream Service Errors

<!-- Code for simple example activity that calls an generic payments API and returns the result (modeled after Stripe) -->
```go
```

---

## Activities: Avoid Amplifying Invalid Requests

<!-- Updated code example that inspects http status code and returns non-retryable TemporalApplicationError for bad requests (HTTP 400) (use switch statement for HTTP status so more cases can easily be added in the future) -->
```go
```

---

## Activities: Avoid Overloading Services With Retries

<!-- Updated code example that also increases the next retry backoff time when external service returns a resource overloaded error (HTTP 429) using the activityhelpers.GetNextRetryDelay function and multiplying its return value by 1.5. -->
```go
```

---

## Activities: Implementing Idempotency

Three techniques to achieve idempotency:
- Passing idempotency key to external APIs
    - Derive a key from the Workflow ID and activity ID. Pass this to downstream systems (like Stripe) to ignore duplicate requests.
- Applying database constraints
    - Use `INSERT ... ON CONFLICT` or conditional writes to ensure records aren't created twice.
- Using naturally idempotent operations
    - Design side effects as state settings (Set to X) rather than increments (+1), or use upserts with fixed IDs.
    - May help to decompose into multiple activities.

## Activities: Natural Idempotency

<!-- New code example: An activity called RunKubernetesJob that starts a k8s job and waits for it to complete (two k8s API calls). The activity should accept a struct with Name and Namespace fields, and return a struct with a Status field (completed or failed) -->
```go
```

<!-- Speaker note: This activity is currently not idempotent because if the second API call fails, then when it's retried it will fail attempting to create a job with the same name instead of attaching to the existing one. -->

<!-- Update code example: Split into two activities, one called StartKubernetesJob and another called AwaitKubernetesJob. -->
```go
```

<!-- Speaker note: Now the workflow needs to call both activities, one after the other, but it doesn't matter how many times either of them is retried and we get more visibility into what's going on through the workflow history. -->

---

## Activities: Idempotency Keys

<!-- Back to the previous payment code example. Update the code to compute an idempotency key (using activityhelpers.GetIdempotencyToken) and add it to the request header (follow the example from stripe docs: https://docs.stripe.com/api/idempotent_requests). -->
```go
```

<!-- Speaker note: if you are lucky enough to be using an API that directly supports idempotency keys, whether in the form of a header or a client side request identifier, then you can also compute one based on the workflow run id and activity id. -->

---

## Activities: Weathering System Outages

<!-- New code example, this time showing the workflow code that invokes the payment activity. Set a 30s start to close timeout and a 1m schedule to close timeout. -->
```go
```

<!-- Speaker note: Temporal is great at retrying activities, but it's still important to think carefully about how we configure timeouts and retry policies in order to survive worst case system outages. For example here I'm invoking my activity with a schedule to close timeout that doesn't give much room for the activity to be retried if the worker or downstream API are temporarily unavailble. -->

<!-- Update the schedule to close timeout to 1h in the code example. Include code comment saying "allow retrying for up to 1 hour".

Speaker note: So the first timeout mistake to avoid is a schedule to close timeout that's too short. Pick a value based on how long you want to retry in the face of a serious system outage.
-->

---

## Activities: Weathering System Outages

<!-- Update code example, now adding a retry policy with MaxAttempts set to 3, initial backoff to 1s, backoff coefficient to 2, and max backoff to 30s. -->
```go
```

<!-- Speaker note: Another way an activity's retry behavior can be unexpectedly limited is by setting MaxAttempts. For example now if this activity quickly returns an error, we would exhaust all of our retries in less than 10 seconds, even though our intent was to survive outages of up to 1 hour. -->

<!-- Comment out the MaxAttempts field in the code example. Include code comment saying "allow unlimited attempts until the ScheduleToClose timeout is reached".

Speaker note: I think a much simpler mental model is to allow unlimited attempts, and only use retry policy to tune the backoff behavior like initial interval and maximum backoff duration as needed.
-->

---

## Activities: Handling Worker Disruptions

Choosing between StartToClose and Heartbeat timeouts

<!-- New workflow code example, this time invoking our (longer running) AwaitKubernetesJob activity. Set a 30s start to close timeout and a 1h schedule to close timeout. -->
```go
```

<!-- Speaker note: Our previous activity example was expected to always complete in under 30s. But that's not the case for all activities. A short start to close timeout made sense for the previous use case, but what about for an activity that could run for much longer? Setting too short a value could mean that some requests never complete, no matter how many retry attempts are made. -->

<!-- Updated code example: Change the start to close timeout to 5m.

Speaker note:
So we can try increasing the start to close timeout, but now this also means that if the worker crashes or becomes unresponsive, we'd have to wait much longer before Temporal retries the activity.
-->

<!-- Updated code example: Replace the start to close timeout with a 30s heartbeat timeout.

Speaker notes:
- Replacing a start to close timeout with heartbeat timeout avoids the tradeoff between retrying quickly when the worker fails, and allowing your longest-running tasks to complete. Now the activity can run as long as it needs to, up to the schedule to close timeout, but is retried quickly if the worker becomes unresponsive.
 -->

---

## Overview of Activity Timeouts & Retry Policy

<!-- 1. Screenshot of StartActivityOptions (every timeout & retry policy field) from the Go SDK -->
```go
```

<!-- 2. Highlight the options in the screenshot that we'll talk about in subsequent slides -->

---

## Activities: 




















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
