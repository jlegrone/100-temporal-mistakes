package testsuite_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
)

func replayTestWorkflow(_ workflow.Context) error { return nil }

func TestAssertWorkflowReplayFromJSONFiles(t *testing.T) {
	c, taskQueue := testsuite.StartDevServerWorker(t, func(r worker.Registry) {
		r.RegisterWorkflow(replayTestWorkflow)
	})

	dir := t.TempDir()
	paths := make([]string, 2)
	for i := range paths {
		run, err := c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{TaskQueue: taskQueue}, replayTestWorkflow)
		if err != nil {
			t.Fatal(err)
		}
		if err := run.Get(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, fmt.Sprintf("history-%d.json", i))
		writeHistoryJSON(t, c, run, path)
		paths[i] = path
	}

	testsuite.AssertWorkflowReplayFromJSONFiles(t, replayTestWorkflow, paths)
}

func writeHistoryJSON(t *testing.T, c client.Client, run client.WorkflowRun, path string) {
	t.Helper()
	iter := c.GetWorkflowHistory(t.Context(), run.GetID(), run.GetRunID(), false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var events []json.RawMessage
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			t.Fatal(err)
		}
		b, err := protojson.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, b)
	}
	out, err := json.MarshalIndent(map[string]any{"events": events}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}
