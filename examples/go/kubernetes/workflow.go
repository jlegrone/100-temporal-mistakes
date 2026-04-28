package kubernetes

import (
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/workflowhelpers"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
)

// RunKubernetesJobRequest is the input to RunKubernetesJob.
type RunKubernetesJobRequest struct {
	Name      string
	Namespace string
	Spec      batchv1.JobSpec
}

// RunKubernetesJobResponse is the output of RunKubernetesJob.
type RunKubernetesJobResponse struct {
	UID    types.UID
	Status string // "completed" or "failed"
}

// RunKubernetesJob starts a Job and awaits its terminal state.
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
	}

	startResp, err := workflowhelpers.AwaitActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			// This activity should return quickly, so no heartbeats needed.
			StartToCloseTimeout:    30 * time.Second,
			ScheduleToCloseTimeout: time.Hour,
			RetryPolicy:            retryPolicy,
		}),
		w.StartKubernetesJob,
		StartKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace, Spec: req.Spec})
	if err != nil {
		return nil, err
	}

	awaitResp, err := workflowhelpers.AwaitActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			// This activity may run for a long time, so use Heartbeat instead of StartToClose timeout.
			HeartbeatTimeout:       30 * time.Second,
			ScheduleToCloseTimeout: time.Hour,
			RetryPolicy:            retryPolicy,
		}),
		w.AwaitKubernetesJob,
		AwaitKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace},
	)
	if err != nil {
		return nil, err
	}
	return &RunKubernetesJobResponse{UID: startResp.UID, Status: awaitResp.Status}, nil
}
