package not_using_workflow_replay_for_debugging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/worker"
)

// @@@SNIPSTART not-using-workflow-replay-for-debugging-good

func TestReplayWorkflow(t *testing.T) {
	// TODO: hook up logs to the test output (add a new workflow replayer helper to the internal testsuite package to do this, it should accept two args (testing.TB and the workflow function to replay)). The function should return the replayer.
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(MyWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "testdata/history.json")
	require.NoError(t, err)
}

// @@@SNIPEND
