# Setting Too-Short Timeouts

> [!TIP]
> * Timeouts should account for downstream failures and temporary outages, not just the happy path execution time.
> * A timeout that's too short defeats Temporal's retry mechanism -- the activity fails permanently before retries have a chance to succeed.
> * Set timeouts based on how long you're willing to wait for eventual success, not how long the operation normally takes.

## What?

A common pattern when configuring activity and workflow timeouts is to base them on expected execution time with a small margin. If an activity normally completes in 2 seconds, you set the timeout to 5 seconds. If a workflow normally runs for 10 minutes, you set the [execution timeout](terms/workflow-execution-timeout.md) to 15 minutes.

This seems reasonable but ignores a fundamental reality: the systems your workflows interact with will experience downtime. Deployments, infrastructure issues, database failovers, rate limiting -- these events can make downstream services unavailable for minutes or even hours. A timeout based on happy-path execution time will expire long before the downstream service recovers, causing your workflow or activity to fail permanently.

## Why?

Temporal's core value is durability through retries and persistence. When a downstream service goes down for 5 minutes during a deployment, Temporal can keep retrying the activity with backoff until the service comes back. The activity eventually succeeds, the workflow continues, and nobody has to intervene.

But this only works if the timeouts give the retries enough room to breathe. Consider this scenario:
- Your activity calls a payment service that normally responds in 1 second.
- You set a [schedule-to-close timeout](terms/schedule-to-close-timeout.md) of 30 seconds.
- The payment service has a rolling deployment that causes 3 minutes of intermittent failures.
- Your activity retries a few times, hits the 30-second schedule-to-close timeout, and fails permanently.
- Now you have a failed workflow that needs manual intervention, even though the payment service is perfectly healthy 3 minutes later.

With a schedule-to-close timeout of 10 minutes, the same scenario plays out differently: the activity retries through the deployment window, succeeds on the other side, and the workflow completes without anyone noticing there was a problem.

Too-short timeouts turn Temporal into a system that amplifies failures rather than absorbing them. Every downstream hiccup becomes a workflow failure that requires attention.

## How?

**Base schedule-to-close timeouts on outage tolerance, not execution time.** Ask yourself: "How long am I willing to wait for this operation to eventually succeed?" If the answer is "I'd rather wait 30 minutes than deal with a failed workflow," set the timeout to 30 minutes. The activity will still complete in 2 seconds under normal conditions.

**Keep [start-to-close](terms/start-to-close-timeout.md) timeouts shorter but reasonable.** The start-to-close timeout limits individual attempts. This can be tighter -- maybe 30 seconds for an API call that normally takes 1 second -- because retries will handle individual attempt failures. The schedule-to-close timeout is what needs to account for prolonged outages.

**Set workflow execution timeouts with even more margin.** A workflow execution timeout should account for the worst case across all activities and any waiting periods. If your workflow has 5 activities that each might need 10 minutes of retries during an outage, your workflow timeout should be well beyond 50 minutes. See also [not setting a workflow timeout](not-setting-a-workflow-timeout.md) -- you should still set one, just don't make it too tight.

**Revisit timeouts when your infrastructure changes.** If you move from a zero-downtime deployment strategy to rolling deployments, your downstream services now have deployment windows where they're partially unavailable. Your timeouts need to reflect this.
