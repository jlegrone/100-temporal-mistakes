# Preventing Activity Retries

<!-- TODO: #presentation-include -->

> [!TIP]
> Three common timeout misconfigurations can silently prevent activities from retrying: no [heartbeat timeout](terms/heartbeat-timeout.md), no [start-to-close timeout](terms/start-to-close-timeout.md), or start-to-close equal to [schedule-to-close](terms/schedule-to-close-timeout.md).

**No start-to-close timeout AND no heartbeat timeout**: Without a heartbeat timeout, Temporal can't detect an unresponsive worker. Instead Temporal will wait the full start-to-close duration before a retry. But if no start to close timeout is set either, then an unresponsive worker will cause the entire schedule to close timeout to elapse without a single retry.

Set a heartbeat timeout so that the activity can be retried more quickly if the worker dies or is redeployed while the activity is running.

**Start-to-close equal to schedule-to-close**: If both are 30 seconds, the first attempt uses the full budget and the schedule-to-close expires simultaneously -- no time remains for a second attempt. Schedule-to-close should be significantly larger, at minimum `start-to-close * max_attempts` plus margin for backoff. Better yet, base it on how long you're willing to wait for eventual success.

<!-- TODO: Add case of retry policy having backoff config that prevents scheduling retries within the schedule to close window. -->

See also: [Setting Too-Short Timeouts](setting-too-short-timeouts.md).

<!-- TODO: Add guidance for presentation slides:
1. Always set schedule to close timeout. This is how long the activity will keep trying during an outage/incident.
2. Always set a heartbeat OR start to close timeout. This is how long the Temporal server should wait before the retry policy kicks in if the worker becomes unresponsive.
3. Be careful not to set start to close timeouts that are too short for your actual business logic.
4. Consider auto-heartbeating. You will be responsive to both worker disruptions AND workflow cancelation this way.
5. Consider requiring schedule to close AND heartbeat timeouts to be set for every activity (provide interceptor example copied from my dispatch agents repo).
 -->
