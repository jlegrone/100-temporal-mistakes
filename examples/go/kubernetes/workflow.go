package kubernetes

import (
	"time"

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
		MaximumInterval:    30 * time.Second,
	}

	startCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Hour,
		RetryPolicy:            retryPolicy,
	})
	startReq := StartKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace, Spec: req.Spec}
	var startResp StartKubernetesJobResponse
	if err := workflow.ExecuteActivity(startCtx, w.StartKubernetesJob, startReq).Get(startCtx, &startResp); err != nil {
		return nil, err
	}

	awaitCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		HeartbeatTimeout:       30 * time.Second,
		ScheduleToCloseTimeout: time.Hour,
		RetryPolicy:            retryPolicy,
	})
	awaitReq := AwaitKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace}
	var awaitResp AwaitKubernetesJobResponse
	if err := workflow.ExecuteActivity(awaitCtx, w.AwaitKubernetesJob, awaitReq).Get(awaitCtx, &awaitResp); err != nil {
		return nil, err
	}
	return &RunKubernetesJobResponse{UID: startResp.UID, Status: awaitResp.Status}, nil
}
