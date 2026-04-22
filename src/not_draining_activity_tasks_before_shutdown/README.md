# Not Draining Activity Tasks Before Shutdown

<!-- Make sure your activity drain configuration is aligned with the lifecycle of your infrastructure, eg. the terminatationGracePeriodSeconds in Kubernetes. Otherwise it will be for naught! -->

> [!TIP]
> When a [worker](../terms/worker.md) is killed during deployment, in-flight activities are abandoned. Configure a graceful shutdown drain period so activities can finish or checkpoint via [heartbeats](../terms/heartbeat.md).

During deployments, workers are stopped and replaced. Without a drain period, the server doesn't know activities failed until their timeout expires, then retries from scratch -- wasting all completed work. This is especially painful for long-running activities where losing minutes of progress per deployment adds up.

<!--SNIPSTART not-draining-activity-tasks-example-->
[not_draining_activity_tasks_before_shutdown/worker.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_draining_activity_tasks_before_shutdown/worker.go)
```go
func NewWorkerWithGracefulStop(c client.Client) worker.Worker {
	return worker.New(c, "my-task-queue", worker.Options{
		WorkerStopTimeout: 5 * time.Minute,
	})
}

```
<!--SNIPEND-->

Make sure your deployment orchestrator (Kubernetes `terminationGracePeriodSeconds`, ECS stop timeout) gives the worker at least as much time as the drain period. Activities that [heartbeat](../terms/heartbeat.md) detect [cancellation](../terms/cancelation.md) (triggered by shutdown) and save progress, so they resume from the last checkpoint on retry rather than starting over.

Match the drain timeout to your longest reasonable activity duration. Activities that run for hours should already be heartbeating and checkpointing, so a shorter drain is fine as long as they can exit promptly.

See also: [Not Using Activity Heartbeat Details](../not_using_activity_heartbeat_details/).

<!--
Old internal blog post about this:

tl;dr We now attempt to drain workers of requests by waiting for in-progress activities to complete instead of exiting immediately when Kubernetes signals it wants to terminate a worker pod. This results in improved stability & workflow execution latency during worker deployments and down-scale events.

The Problem
The Temporal server schedules activity retries when it detects that either of two types of timeouts have been exceeded: Heartbeat and StartToClose.

Any time an activity is retried, this adds latency to the overall workflow execution. How much latency depends on how long the next activity attempt takes to complete and the duration of the activity's retry backoff.

An activity may also be marked as failed instead of retrying if it has already reached its max retry count or exhausted its ScheduleToClose timeout. These types of errors usually bubble up and cause workflow failures.

For more background, see this blog post from Temporal: The four types of Activity timeouts 

Workers are routinely terminated, most often during a worker deployment or because of a downscale event triggered by the WatermarkPodAutoscaler.

Before, when workers terminated we would immediately cancel all running activity contexts and rely on Temporal to retry these activities by scheduling them on another worker process. As noted above, this increases latency and eats into our retry budget.

In practice we've never noticed issues with this behavior in production (well, aside from local activities during the incident!). But as more teams onboard to deploy trains and we work to optimize costs through horizontal autoscaling, we want to ensure that additional pod churn doesn't negatively impact the performance or error rate of workflow executions.

A Solution
Thankfully the Temporal Go SDK already provides an option which we had overlooked so far: WorkerStopTimeout. When set, the worker's Stop() function will block for up to this duration while waiting for in-progress activities to complete.

By avoiding activity retries we should be able to improve on our workflow execution tail latency!

Validation
To test the theory behind these changes, we came up with an experiment where we would deploy the heartbeat-worker service with an updated SendHeartbeatMetric activity that sleeps for 10 seconds:

func (w *Worker) SendHeartbeatMetric(ctx context.Context) error {
   // Simulate a long(ish) running activity
   time.Sleep(10 * time.Second)
   return nil  
}
We then applied load by starting 10,000 heartbeat workflow executions over a period of about 5 minutes, repeating this procedure three times.

1. Baseline
For the first load test, we ran the worker without any disruptions. As expected, a distribution graph of the temporal_heartbeat_workflow_duration metric lines up with the value of time.Sleep() in our activity function:


2. Worst case pre-incident
For the second test, we wanted to see what the worst case behavior was before the fixes applied in dd-source/pull/40664.

To simulate a situation where pods are frequently terminating, we ran this script while applying load:



while true; do kubectl -n atlas-dev rollout restart deploy heartbeat; sleep 30; done
Note that this is an extreme scenario; pods end up in a running state for a maximum of around 90 seconds before being terminated. In production pods should generally cycle out much more slowly due to pod disruption budgets, minReadySeconds, and WatermarkPodAutoscaler's scaleDownLimitFactor / downscaleForbiddenWindowSeconds configuration.

75% of our requests are still able to complete in 10 seconds without retries. But we also see that 10% take over 40 seconds. 😱


3. Worst case post-incident
Finally we ran a test with the fixes from the incident, using the same disruption script.

Our p99 went from 46 seconds to 15! And only 5 requests out of 10,000 took longer than 16 seconds.
 -->
