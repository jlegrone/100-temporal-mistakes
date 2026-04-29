package testhelpers

import (
	"testing"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/jlegrone/100-temporal-mistakes/examples/go/activitypolicyinterceptor"
)

// ReplayWorkflowHistoryFromJSONFile replays a recorded workflow history (from
// a JSON file) against the provided workflow function. Workflow logs are
// routed to t.Log so they appear in `go test -v` output, including logs
// emitted during replay.
//
// The Activity Policy Interceptor is installed at its strictest defaults; pass
// [WithActivityPolicySeverity] to lower or silence individual policies for
// replays of histories that intentionally violate the policies (e.g. tests
// demonstrating non-compliant patterns).
func ReplayWorkflowHistoryFromJSONFile(t TB, workflowFn any, historyPath string, opts ...HelperOption) error {
	t.Helper()
	replayer := newWorkflowHistoryReplayer(t, workflowFn, opts)
	return replayer.ReplayWorkflowHistoryFromJSONFile(&tLogger{t: t}, historyPath)
}

// AssertWorkflowReplayFromJSONFiles replays each named workflow history
// against the provided workflow function as a subtest. The file names are used
// as the subtest names.
//
// The Activity Policy Interceptor is installed at its strictest defaults; pass
// [WithActivityPolicySeverity] to lower or silence individual policies for
// replays of histories that intentionally violate the policies.
func AssertWorkflowReplayFromJSONFiles(t *testing.T, workflowFn any, historyFiles []string, opts ...HelperOption) {
	t.Helper()
	for _, historyFile := range historyFiles {
		t.Run(historyFile, func(t *testing.T) {
			t.Helper()
			if err := ReplayWorkflowHistoryFromJSONFile(t, workflowFn, historyFile, opts...); err != nil {
				t.Error(err)
			}
		})
	}
}

func newWorkflowHistoryReplayer(t TB, workflowFn any, opts []HelperOption) worker.WorkflowReplayer {
	t.Helper()
	cfg := newHelperConfig(opts)
	replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		// Set the default data converter explicitly so the SDK doesn't log
		// "No DataConverter configured" on every test run.
		DataConverter: converter.GetDefaultDataConverter(),
		// The SDK suppresses workflow logger output during replay by default
		// to avoid duplicate log lines. We're replaying for test output, so
		// enable it.
		EnableLoggingInReplay: true,
		// Install the Activity Policy Interceptor at its strictest defaults
		// (overridable per-policy via WithActivityPolicySeverity).
		Interceptors: []interceptor.WorkerInterceptor{
			activitypolicy.New(cfg.activityPolicyOptions()),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	replayer.RegisterWorkflow(workflowFn)
	return replayer
}
