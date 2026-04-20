package internal

import (
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/testsuite"
)

// TB is the subset of testing.TB that both *testing.T and *rapid.T satisfy.
type TB interface {
	Helper()
	Log(args ...any)
}

// NewTestWorkflowEnvironment creates a test workflow environment with
// logs directed to t.Log so they appear in go test -v output.
func NewTestWorkflowEnvironment(t TB) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	suite.SetLogger(&tLogger{t: t})
	return suite.NewTestWorkflowEnvironment()
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
