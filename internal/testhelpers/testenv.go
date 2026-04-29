package testhelpers

import (
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	sdktestsuite "go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/jlegrone/100-temporal-mistakes/examples/go/activitypolicyinterceptor"
)

// TB is the subset of testing.TB that both *testing.T and *rapid.T satisfy.
type TB interface {
	Helper()
	Log(args ...any)
	Fatal(args ...any)
}

// NewTestWorkflowEnvironment creates a test workflow environment with logs
// directed to t.Log so they appear in go test -v output. The Activity Policy
// Interceptor is installed at its strictest defaults (every policy at
// SeverityError, AutoHeartbeat enabled), so any test that schedules an
// activity in violation of the spec will surface a PolicyViolationError.
//
// Tests that need to demonstrate the issues the policy interceptor is designed
// to catch can clear (or override) the worker options on the returned env via
// `env.SetWorkerOptions(worker.Options{})`.
func NewTestWorkflowEnvironment(t TB) *sdktestsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &sdktestsuite.WorkflowTestSuite{}
	suite.SetLogger(&tLogger{t: t})
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			activitypolicy.New(activitypolicy.Options{}),
		},
	})
	return env
}

type tLogger struct {
	t TB
}

var _ log.Logger = (*tLogger)(nil)

func (l *tLogger) Debug(msg string, keyvals ...any) {
	l.t.Helper()
	l.t.Log(append([]any{"DEBUG", msg}, keyvals...)...)
}

func (l *tLogger) Info(msg string, keyvals ...any) {
	l.t.Helper()
	l.t.Log(append([]any{"INFO", msg}, keyvals...)...)
}

func (l *tLogger) Warn(msg string, keyvals ...any) {
	l.t.Helper()
	l.t.Log(append([]any{"WARN", msg}, keyvals...)...)
}

func (l *tLogger) Error(msg string, keyvals ...any) {
	l.t.Helper()
	l.t.Log(append([]any{"ERROR", msg}, keyvals...)...)
}
