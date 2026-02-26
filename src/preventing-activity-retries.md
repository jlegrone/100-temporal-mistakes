# Preventing Activity Retries

> [!TIP]
> * Without a [heartbeat timeout](terms/heartbeat-timeout.md), Temporal can't detect a stuck activity and won't retry it until the start-to-close timeout expires.
> * Without a [start-to-close timeout](terms/start-to-close-timeout.md), you have no per-attempt timeout -- a single hung attempt blocks the activity for the entire schedule-to-close duration.
> * Setting start-to-close equal to [schedule-to-close](terms/schedule-to-close-timeout.md) effectively gives you only one attempt since both expire at the same time.

## What?

Temporal's activity retry mechanism is one of its most powerful features, but it only works well when you configure timeouts correctly. There are three common timeout misconfigurations that silently prevent activities from retrying as expected.

### Not setting a heartbeat timeout

Long-running activities should report [heartbeats](terms/heartbeat.md) to let the server know they're still making progress. Without a heartbeat timeout, Temporal has no way to detect that an activity is stuck -- say, the [worker](terms/worker.md) process is deadlocked, the network connection is hanging, or the activity is blocked on an unresponsive downstream service. Temporal will wait for the entire start-to-close timeout to expire before considering the attempt failed and retrying. If your activity normally completes in 10 seconds but gets stuck, you'll wait the full start-to-close duration (potentially minutes or hours) before a retry kicks in.

### Not setting a start-to-close timeout

The start-to-close timeout limits how long a single activity attempt can run. If you only set a schedule-to-close timeout, there is no per-attempt limit. A single attempt can consume the entire schedule-to-close budget, leaving no room for retries. This effectively turns your activity into a one-shot execution with a deadline rather than a retriable operation.

### Setting start-to-close equal to schedule-to-close

This is a subtle but common mistake. The schedule-to-close timeout is the overall deadline across all attempts, while start-to-close is the deadline for each individual attempt. If both are set to 30 seconds, the first attempt runs for up to 30 seconds, at which point the schedule-to-close timeout has also expired. There's no time budget left for a second attempt.

## Why?

All three of these misconfigurations have the same effect: they prevent Temporal from doing what it does best, which is automatically retrying failed operations. Instead of getting fast recovery from transient failures, you end up with slow failures, no retries, or both.

This matters most in production when downstream services have intermittent issues. A properly configured activity might detect a stuck attempt via heartbeat timeout in 30 seconds, retry, and succeed on the second attempt -- total time under a minute. A misconfigured one might wait 10 minutes for the start-to-close timeout, then fail because the schedule-to-close timeout is also 10 minutes. No retry, just a slow failure.

## How?

**Set a heartbeat timeout** on any activity that runs longer than a few seconds. The heartbeat timeout should be a small multiple of your expected heartbeat interval. If you heartbeat every 10 seconds, set the heartbeat timeout to 30-60 seconds.

**Always set a start-to-close timeout** that reflects the maximum duration of a single attempt. This should be generous enough to handle slow responses but short enough to allow multiple retries within your schedule-to-close window.

**Set schedule-to-close to be significantly larger than start-to-close.** As a rule of thumb, schedule-to-close should be at least `start-to-close * max_attempts` plus some margin. If you want 3 retries with a 1-minute start-to-close timeout, set schedule-to-close to at least 5 minutes. Better yet, set schedule-to-close based on how long you're willing to wait for the activity to eventually succeed, factoring in backoff delays between retries.
