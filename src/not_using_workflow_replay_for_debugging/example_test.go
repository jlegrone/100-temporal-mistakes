package not_using_workflow_replay_for_debugging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/worker"
)

// YourWorkflow is a placeholder for the workflow function to replay.
func YourWorkflow() {}

// @@@SNIPSTART not-using-workflow-replay-for-debugging-good

func TestReplayWorkflow(t *testing.T) {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(YourWorkflow)
	err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "history.json")
	require.NoError(t, err)
}

// @@@SNIPEND
