# Setting Too-Short Timeouts

> [!TIP]
> Set timeouts based on how long you're willing to wait for eventual success, not how long the operation normally takes. Too-short timeouts defeat Temporal's retry mechanism.

A common pattern is to base timeouts on happy-path execution time with a small margin -- an activity that completes in 2 seconds gets a 5-second timeout. This ignores a fundamental reality: downstream services experience downtime. Deployments, infrastructure issues, and rate limiting can make services unavailable for minutes or hours. A timeout based on happy-path execution expires long before the service recovers, causing the activity to fail permanently even though it would have succeeded minutes later.

Consider: your activity calls a payment service (normally 1 second), with a 30-second [schedule-to-close timeout](terms/schedule-to-close-timeout.md). A rolling deployment causes 3 minutes of intermittent failures. The activity retries a few times, hits the timeout, and fails permanently. With a 10-minute timeout, the same scenario plays out differently: retries succeed after the deployment window and nobody notices.

Base schedule-to-close timeouts on outage tolerance ("how long am I willing to wait?"). Keep [start-to-close](terms/start-to-close-timeout.md) timeouts tighter since retries handle individual attempt failures. Set [workflow execution timeouts](terms/workflow-execution-timeout.md) with even more margin to account for the worst case across all activities.

See also: [Not Setting a Workflow Timeout](not-setting-a-workflow-timeout.md), [Preventing Activity Retries](preventing-activity-retries.md).
