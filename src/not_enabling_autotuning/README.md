# Not Enabling Worker Autotuning

<!-- Autotuning is great! Start with that, and only explore more explicit tuning settings if you have identified bottlenecks specific to your workload. With autotuning, in theory it is also possible to simplify horizontal autoscaling by scaling in and out based on resource usage on the worker process. -->

> [!TIP]
> Temporal SDKs offer autotuning that automatically adjusts [worker](../terms/worker.md) concurrency settings based on system resource utilization, replacing error-prone manual tuning of values like `MaxConcurrentActivities`.

Temporal workers have several concurrency knobs: maximum concurrent [workflow tasks](../terms/workflow-task.md), maximum concurrent activities, number of pollers, and more. By default, these are set to fixed values that represent conservative guesses. Most teams either leave the defaults untouched or manually set them based on load testing, then never revisit them. Manual tuning is problematic: conservative settings leave capacity on the table, while aggressive settings risk overwhelming the worker. Load characteristics also change over time as worker code is modified and traffic patterns evolve.

Worker autotuning replaces this static configuration with a dynamic feedback loop. It adjusts concurrency at runtime based on actual CPU and memory usage, ramping up when resources are available and backing off under pressure:

<!-- TODO: make sure this is up to date, and hardcode an infosupplier instead of leaving that up to the imagination. -->
<!--SNIPSTART not-enabling-autotuning-good-->
[not_enabling_autotuning/worker.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_enabling_autotuning/worker.go)
```go

// Good: use resource-based autotuning instead of static concurrency settings.
func NewWorkerOptions(infoSupplier worker.SysInfoProvider) (worker.Options, error) {
	tuner, err := worker.NewResourceBasedTuner(worker.ResourceBasedTunerOptions{
		TargetCpu:    0.8,
		TargetMem:    0.8,
		InfoSupplier: infoSupplier,
	})
	if err != nil {
		return worker.Options{}, err
	}
	return worker.Options{
		Tuner: tuner,
	}, nil
}

```
<!--SNIPEND-->

<!-- TODO: Describe which specific worker/SDK metrics to monitor and what their target values should look like. For one thing, you probably want to see that autotuning is getting you close to your utilization targets configured in worker options. -->
Set target resource utilization thresholds that make sense for your environment, and monitor after enabling autotuning via SDK metrics.
