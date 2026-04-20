# Not Enabling Worker Autotuning

> [!TIP]
> Temporal SDKs offer autotuning that automatically adjusts [worker](../terms/worker.md) concurrency settings based on system resource utilization, replacing error-prone manual tuning of values like `MaxConcurrentActivities`.

Temporal workers have several concurrency knobs: maximum concurrent [workflow tasks](../terms/workflow-task.md), maximum concurrent activities, number of pollers, and more. By default, these are set to fixed values that represent conservative guesses. Most teams either leave the defaults untouched or manually set them based on load testing, then never revisit them. Manual tuning is problematic: conservative settings leave capacity on the table, aggressive settings risk overwhelming the host with missed [heartbeats](../terms/heartbeat.md) and cascading failures, and load characteristics change over time as new activity types, input sizes, and traffic patterns evolve.

Worker autotuning (currently in preview) replaces this static guess with a dynamic feedback loop. It adjusts concurrency at runtime based on actual CPU and memory usage, ramping up when resources are available and backing off under pressure:

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

Set target resource utilization thresholds that make sense for your environment, and monitor the actual concurrency after enabling autotuning via SDK metrics showing current concurrent activities and workflow tasks. Be aware of the preview status -- test thoroughly in non-production environments before rolling out widely, and watch SDK release notes for changes to the API or behavior.
