# Not Enabling Worker Autotuning

> [!TIP]
> * Temporal SDKs offer autotuning (preview) that automatically adjusts [worker](terms/worker.md) concurrency settings based on system resource utilization.
> * Without autotuning, you must manually tune values like `MaxConcurrentActivities` -- which is error-prone and doesn't adapt to changing load.
> * Enabling autotuning lets workers use more of their available capacity without the risk of over-committing resources.

## What?

Temporal workers have several concurrency knobs: maximum concurrent [workflow tasks](terms/workflow-task.md), maximum concurrent activities, number of pollers, and more. By default, these are set to fixed values that represent conservative guesses. Most teams either leave the defaults untouched or manually set them based on load testing, then never revisit them.

Worker autotuning is a feature (currently in preview) that automatically adjusts these concurrency settings at runtime based on actual system resource usage (CPU, memory). Instead of committing to a fixed number like "this worker can run 50 concurrent activities", autotuning lets the worker discover its own capacity: it ramps up concurrency when resources are available and backs off when the system is under pressure.

## Why?

Manual tuning of worker concurrency is problematic:

- **Underprovisioning**: Conservative settings leave worker capacity on the table. A worker capable of running 100 concurrent activities but configured for 20 is wasting 80% of its potential throughput.
- **Overprovisioning**: Aggressive settings risk overwhelming the host. If you set `MaxConcurrentActivities` too high and those activities are CPU-intensive, the worker can become unresponsive, miss [heartbeats](terms/heartbeat.md), and cause cascading failures.
- **Static configuration**: Load characteristics change over time -- new activity types, different input sizes, varying traffic patterns. A value that was correct last month may be wrong today.
- **Operational burden**: Manually tuning workers across many [task queues](terms/task-queue.md) and activity types is tedious. Teams often skip it, leaving suboptimal defaults in place.

Autotuning addresses all of these by replacing a static guess with a dynamic feedback loop.

## How?

1. **Enable autotuning in your worker configuration**. The exact API depends on the SDK. For example, in the Go SDK, you can configure a `ResourceBasedTuner` that monitors CPU and memory usage:

   ```go
   tuner, _ := worker.NewResourceBasedTuner(worker.ResourceBasedTunerOptions{
       TargetCPUUsage:    0.8,
       TargetMemoryUsage: 0.8,
   })
   workerOptions := worker.Options{
       Tuner: tuner,
   }
   ```

2. **Set target resource utilization thresholds** that make sense for your environment. The defaults are typically conservative (e.g. 80% CPU, 80% memory). You can adjust these based on your tolerance and whether the worker shares the host with other processes.

3. **Monitor the actual concurrency** after enabling autotuning. The SDKs expose metrics showing the current number of concurrent activities and workflow tasks. Verify that autotuning is behaving as expected -- ramping up under load and backing off when resources are scarce.

4. **Be aware of the preview status**. Autotuning is not yet GA in all SDKs. Test it thoroughly in non-production environments before rolling it out widely, and keep an eye on SDK release notes for changes to the API or behavior.
