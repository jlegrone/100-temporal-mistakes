package testdata

import (
	"encoding/json"
	"os"
	"testing"

	parent "github.com/jlegrone/100-temporal-mistakes/src/not_using_workflow_replay_for_debugging"

	internaltestsuite "github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestGenerateHistory runs the workflow against a dev server and saves
// the history to history.json. Run to regenerate:
//
//	go test ./src/not_using_workflow_replay_for_debugging/testdata/ -run TestGenerateHistory -count=1
func TestGenerateHistory(t *testing.T) {
	c, taskQueue := internaltestsuite.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(parent.MyWorkflow)
		r.RegisterActivity(parent.GreetActivity)
	})

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, parent.MyWorkflow, "World")
	require.NoError(t, err)

	var result string
	require.NoError(t, run.Get(t.Context(), &result))
	require.Equal(t, "Hello, World!", result)

	// Export history to JSON.
	iter := c.GetWorkflowHistory(t.Context(), run.GetID(), run.GetRunID(), false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var events []json.RawMessage
	for iter.HasNext() {
		event, err := iter.Next()
		require.NoError(t, err)
		b, err := protojson.Marshal(event)
		require.NoError(t, err)
		events = append(events, b)
	}

	out, err := json.MarshalIndent(map[string]any{"events": events}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile("history.json", out, 0644))
}
