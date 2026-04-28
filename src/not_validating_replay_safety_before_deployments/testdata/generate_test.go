package testdata

import (
	"encoding/json"
	"os"
	"testing"

	parent "github.com/jlegrone/100-temporal-mistakes/src/not_validating_replay_safety_before_deployments"

	"github.com/jlegrone/100-temporal-mistakes/internal/testhelpers"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestGenerateHistory runs the workflow against a dev server and saves
// the history. Run to regenerate:
//
//	go test ./src/not_validating_replay_safety_before_deployments/testdata/ -run TestGenerateHistory -count=1
func TestGenerateHistory(t *testing.T) {
	c, taskQueue := testhelpers.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(parent.MyWorkflow)
		r.RegisterActivity(parent.MyActivity)
	})

	run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{
		TaskQueue: taskQueue,
	}, parent.MyWorkflow)
	require.NoError(t, err)
	require.NoError(t, run.Get(t.Context(), nil))

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
	require.NoError(t, os.WriteFile("my_workflow_history.json", out, 0644))
}
