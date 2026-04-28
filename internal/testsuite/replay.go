package testsuite

import (
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
)

// ReplayWorkflowHistoryFromJSONFile replays a recorded workflow history (from a JSON file)
// against the provided workflow function. Workflow logs are routed to t.Log so
// they appear in `go test -v` output, including logs emitted during replay.
func ReplayWorkflowHistoryFromJSONFile(t TB, workflowFn any, historyPath string) error {
	t.Helper()
	replayer := newWorkflowHistoryReplayer(t, workflowFn)
	return replayer.ReplayWorkflowHistoryFromJSONFile(&tLogger{t: t}, historyPath)
}

func newWorkflowHistoryReplayer(t TB, workflowFn any) worker.WorkflowReplayer {
	t.Helper()
	replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		// Set the default data converter explicitly so the SDK doesn't log
		// "No DataConverter configured" on every test run.
		DataConverter: converter.GetDefaultDataConverter(),
		// The SDK suppresses workflow logger output during replay by default
		// to avoid duplicate log lines. We're replaying for test output, so
		// enable it.
		EnableLoggingInReplay: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return replayer
}
