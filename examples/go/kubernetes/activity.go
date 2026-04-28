// Package kubernetes contains the final-state code examples for the
// "Activities: Natural Idempotency" and "Activities: Handling Worker
// Disruptions" sections of SLIDES.md. The original RunKubernetesJob activity
// was split into StartKubernetesJob and AwaitKubernetesJob so that retries of
// either step are naturally idempotent.
package kubernetes

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/jlegrone/100-temporal-mistakes/internal/activityhelpers"
)

// StartKubernetesJobRequest is the input to StartKubernetesJob.
type StartKubernetesJobRequest struct {
	Name      string
	Namespace string
	Spec      batchv1.JobSpec
}

// StartKubernetesJobResponse is the output of StartKubernetesJob.
type StartKubernetesJobResponse struct {
	// UID is the server-assigned identity of the Job. It is stable across
	// activity retries: subsequent attempts re-fetch the existing Job and
	// report the same UID.
	UID types.UID
}

// AwaitKubernetesJobRequest is the input to AwaitKubernetesJob.
type AwaitKubernetesJobRequest struct {
	Name      string
	Namespace string
}

// AwaitKubernetesJobResponse is the output of AwaitKubernetesJob.
type AwaitKubernetesJobResponse struct {
	Status string // "completed" or "failed"
}

// Worker groups Kubernetes activities so they can share a configured clientset.
type Worker struct {
	client kubernetes.Interface
}

// NewWorker constructs a Worker that issues Kubernetes API calls through the
// supplied clientset. *kubernetes.Clientset and the fake clientset from
// k8s.io/client-go/kubernetes/fake both satisfy kubernetes.Interface.
func NewWorker(client kubernetes.Interface) *Worker {
	return &Worker{client: client}
}

// StartKubernetesJob creates the Job and returns its server-assigned UID.
// Retries are safe: on AlreadyExists, the activity fetches the existing Job
// and returns its UID, so the workflow sees a stable identity across retries.
func (w *Worker) StartKubernetesJob(ctx context.Context, req StartKubernetesJobRequest) (*StartKubernetesJobResponse, error) {
	jobs := w.client.BatchV1().Jobs(req.Namespace)
	created, err := jobs.Create(ctx, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: req.Spec,
	}, metav1.CreateOptions{})
	switch {
	case err == nil:
		return &StartKubernetesJobResponse{UID: created.UID}, nil
	case apierrors.IsAlreadyExists(err):
		existing, getErr := jobs.Get(ctx, req.Name, metav1.GetOptions{})
		if getErr != nil {
			return nil, temporal.NewApplicationErrorWithCause(
				fmt.Sprintf("get existing job %s/%s: %v", req.Namespace, req.Name, getErr),
				"GetJob", getErr,
			)
		}
		return &StartKubernetesJobResponse{UID: existing.UID}, nil
	default:
		return nil, temporal.NewApplicationErrorWithCause(
			fmt.Sprintf("create job %s/%s: %v", req.Namespace, req.Name, err),
			"CreateJob", err,
		)
	}
}

// AwaitKubernetesJob polls the named Job until it reaches a terminal state.
// Heartbeats are emitted automatically so worker hangs trigger a fast retry
// (governed by the activity's HeartbeatTimeout).
func (w *Worker) AwaitKubernetesJob(ctx context.Context, req AwaitKubernetesJobRequest) (*AwaitKubernetesJobResponse, error) {
	cancel := activityhelpers.AutoHeartbeat(ctx)
	defer cancel()

	jobs := w.client.BatchV1().Jobs(req.Namespace)
	tock := time.Tick(30 * time.Second)

	for {
		j, err := jobs.Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return nil, temporal.NewApplicationErrorWithCause(
				fmt.Sprintf("get job %s/%s: %v", req.Namespace, req.Name, err),
				"GetJob", err,
			)
		}
		if status, done := terminalStatus(j); done {
			return &AwaitKubernetesJobResponse{Status: status}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tock:
			continue
		}
	}
}

// terminalStatus inspects a Job's conditions and returns ("completed", true),
// ("failed", true), or ("", false) when the Job has not yet finished.
func terminalStatus(j *batchv1.Job) (string, bool) {
	for _, cond := range j.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		switch cond.Type {
		case batchv1.JobComplete:
			return "completed", true
		case batchv1.JobFailed:
			return "failed", true
		}
	}
	return "", false
}
