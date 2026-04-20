package not_enabling_autotuning

import (
	"go.temporal.io/sdk/worker"
)

// @@@SNIPSTART not-enabling-autotuning-good

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

// @@@SNIPEND
