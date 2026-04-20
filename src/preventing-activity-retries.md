# Preventing Activity Retries

> [!TIP]
> Three common timeout misconfigurations silently prevent activities from retrying: no [heartbeat timeout](terms/heartbeat-timeout.md), no [start-to-close timeout](terms/start-to-close-timeout.md), or start-to-close equal to [schedule-to-close](terms/schedule-to-close-timeout.md).

**No heartbeat timeout**: Without a heartbeat timeout, Temporal can't detect a stuck activity. If a [worker](terms/worker.md) deadlocks or hangs on an unresponsive service, you wait the full start-to-close duration before a retry. Set a heartbeat timeout on any long-running activity -- a small multiple of your expected [heartbeat](terms/heartbeat.md) interval.

**No start-to-close timeout**: Without a per-attempt limit, a single hung attempt consumes the entire schedule-to-close budget, leaving no room for retries. Always set a start-to-close timeout reflecting the maximum duration of a single attempt.

**Start-to-close equal to schedule-to-close**: If both are 30 seconds, the first attempt uses the full budget and the schedule-to-close expires simultaneously -- no time remains for a second attempt. Schedule-to-close should be significantly larger, at minimum `start-to-close * max_attempts` plus margin for backoff. Better yet, base it on how long you're willing to wait for eventual success.

See also: [Setting Too-Short Timeouts](setting-too-short-timeouts.md).
