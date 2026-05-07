package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newActivityEnv() *testsuite.TestActivityEnvironment {
	suite := &testsuite.WorkflowTestSuite{}
	return suite.NewTestActivityEnvironment()
}

func TestStartKubernetesJob_Creates(t *testing.T) {
	cs := fake.NewSimpleClientset()
	w := NewWorker(cs)

	env := newActivityEnv()
	env.RegisterActivity(w.StartKubernetesJob)
	val, err := env.ExecuteActivity(w.StartKubernetesJob, StartKubernetesJobRequest{
		Name:      "etl",
		Namespace: "data",
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:  "main",
						Image: "busybox:1.36",
					}},
				},
			},
		},
	})
	require.NoError(t, err)
	var resp StartKubernetesJobResponse
	require.NoError(t, val.Get(&resp))

	got, err := cs.BatchV1().Jobs("data").Get(t.Context(), "etl", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "etl", got.Name)
	require.Equal(t, "busybox:1.36", got.Spec.Template.Spec.Containers[0].Image)
	require.Equal(t, got.UID, resp.UID)
}

func TestStartKubernetesJob_AlreadyExistsIsIdempotent(t *testing.T) {
	const existingUID types.UID = "11111111-2222-3333-4444-555555555555"
	cs := fake.NewSimpleClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "etl", Namespace: "data", UID: existingUID},
	})
	w := NewWorker(cs)

	env := newActivityEnv()
	env.RegisterActivity(w.StartKubernetesJob)
	val, err := env.ExecuteActivity(w.StartKubernetesJob, StartKubernetesJobRequest{Name: "etl", Namespace: "data"})
	require.NoError(t, err)
	var resp StartKubernetesJobResponse
	require.NoError(t, val.Get(&resp))
	require.Equal(t, existingUID, resp.UID)
}

func TestStartKubernetesJob_PropagatesOtherErrors(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "jobs", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("api server unreachable")
	})
	w := NewWorker(cs)

	env := newActivityEnv()
	env.RegisterActivity(w.StartKubernetesJob)
	_, err := env.ExecuteActivity(w.StartKubernetesJob, StartKubernetesJobRequest{Name: "etl", Namespace: "data"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "api server unreachable")
}

func TestAwaitKubernetesJob_ReturnsCompletedStatus(t *testing.T) {
	cs := fake.NewSimpleClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "etl", Namespace: "data"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
		}},
	})
	w := NewWorker(cs)

	env := newActivityEnv()
	env.RegisterActivity(w.AwaitKubernetesJob)
	val, err := env.ExecuteActivity(w.AwaitKubernetesJob, AwaitKubernetesJobRequest{Name: "etl", Namespace: "data"})
	require.NoError(t, err)

	var resp AwaitKubernetesJobResponse
	require.NoError(t, val.Get(&resp))
	require.Equal(t, "completed", resp.Status)
}

func TestAwaitKubernetesJob_ReturnsFailedStatus(t *testing.T) {
	cs := fake.NewSimpleClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "etl", Namespace: "data"},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
		}},
	})
	w := NewWorker(cs)

	env := newActivityEnv()
	env.RegisterActivity(w.AwaitKubernetesJob)
	val, err := env.ExecuteActivity(w.AwaitKubernetesJob, AwaitKubernetesJobRequest{Name: "etl", Namespace: "data"})
	require.NoError(t, err)

	var resp AwaitKubernetesJobResponse
	require.NoError(t, val.Get(&resp))
	require.Equal(t, "failed", resp.Status)
}
