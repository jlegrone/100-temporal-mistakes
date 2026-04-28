package testhelpers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/api/enums/v1"
	history "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	sdktestsuite "go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
)

func noopActivity(_ context.Context) error { return nil }

// TestTemporalChangeVersion_NotSetWhenPatchEvaluatedDuringReplay confirms
// that hoisting a GetVersion call to the top of a long-running workflow does
// NOT populate the TemporalChangeVersion search attribute on executions that
// were already in flight when the patch was deployed.
//
// Setup:
//   - The workflow registers an update handler that runs a noop activity, then
//     blocks forever on workflow.Await.
//   - A v0 worker (build id "foo") starts the execution and processes the
//     first update.
//   - The v0 worker is stopped; a v1 worker (build id "bar") with an added
//     GetVersion call between the handler registration and the Await takes
//     over and processes a second update.
//
// Validation: WorkflowTaskCompleted events must be stamped by both build IDs,
// proving the execution was genuinely shared between the two worker versions.
//
// Result: GetVersion returns DefaultVersion and the search attribute stays
// unset. When v1 replays the workflow, the handler goroutine still has v0's
// recorded events to consume, so the SDK considers the execution to be in
// replay at the moment GetVersion runs in the main goroutine. No marker is
// written.
func TestTemporalChangeVersion_NotSetWhenPatchEvaluatedDuringReplay(t *testing.T) {
	ctx := t.Context()

	server, err := sdktestsuite.StartDevServer(ctx, sdktestsuite.DevServerOptions{LogLevel: "error"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Stop() })
	c := server.Client()
	t.Cleanup(func() { c.Close() })

	taskQueue := fmt.Sprintf("%s-%s", t.Name(), t.TempDir())
	workflowID := "tcv-experiment"
	patchID := "my-patch"
	activityOpts := workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}

	// v0 workflow: register update handler that runs a noop activity, then
	// block forever.
	v0 := func(ctx workflow.Context) error {
		if err := workflow.SetUpdateHandler(ctx, "Process", func(ctx workflow.Context) error {
			ctx = workflow.WithActivityOptions(ctx, activityOpts)
			return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
		}); err != nil {
			return err
		}
		return workflow.Await(ctx, func() bool { return false })
	}

	w1 := worker.New(c, taskQueue, worker.Options{BuildID: "foo"})
	w1.RegisterWorkflowWithOptions(v0, workflow.RegisterOptions{Name: "MyWorkflow"})
	w1.RegisterActivity(noopActivity)
	if err := w1.Start(); err != nil {
		t.Fatal(err)
	}

	if _, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: taskQueue,
		ID:        workflowID,
	}, "MyWorkflow"); err != nil {
		t.Fatal(err)
	}

	sendUpdate(t, c, workflowID, "update-1")

	dir := t.TempDir()
	captureHistoryToFile(t, c, workflowID, filepath.Join(dir, "history_v0.json"))

	w1.Stop()

	// v1 workflow: same handler, but adds a GetVersion call between the
	// handler registration and the blocking Await.
	var observedVersion workflow.Version
	v1 := func(ctx workflow.Context) error {
		if err := workflow.SetUpdateHandler(ctx, "Process", func(ctx workflow.Context) error {
			ctx = workflow.WithActivityOptions(ctx, activityOpts)
			return workflow.ExecuteActivity(ctx, noopActivity).Get(ctx, nil)
		}); err != nil {
			return err
		}
		observedVersion = workflow.GetVersion(ctx, patchID, workflow.DefaultVersion, 1)
		return workflow.Await(ctx, func() bool { return false })
	}
	w2 := worker.New(c, taskQueue, worker.Options{BuildID: "bar"})
	w2.RegisterWorkflowWithOptions(v1, workflow.RegisterOptions{Name: "MyWorkflow"})
	w2.RegisterActivity(noopActivity)
	if err := w2.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w2.Stop() })

	sendUpdate(t, c, workflowID, "update-2")

	events := captureHistoryToFile(t, c, workflowID, filepath.Join(dir, "history_v1.json"))

	// Validate the test was set up correctly: WorkflowTaskCompleted events
	// must have been stamped by both build IDs.
	seenBuildIDs := map[string]int{}
	for _, ev := range events {
		if attrs := ev.GetWorkflowTaskCompletedEventAttributes(); attrs != nil {
			if wv := attrs.GetWorkerVersion(); wv != nil {
				seenBuildIDs[wv.GetBuildId()]++
			}
		}
	}
	t.Logf("WorkflowTaskCompleted events by build ID: %v", seenBuildIDs)
	if seenBuildIDs["foo"] == 0 {
		t.Fatalf("test setup invalid: no events processed by build ID 'foo'; got %v", seenBuildIDs)
	}
	if seenBuildIDs["bar"] == 0 {
		t.Fatalf("test setup invalid: no events processed by build ID 'bar'; got %v", seenBuildIDs)
	}

	t.Logf("GetVersion returned (last call): %v", observedVersion)
	if observedVersion != workflow.DefaultVersion {
		t.Errorf("expected GetVersion to return DefaultVersion (%d) when called during replay; got %v",
			workflow.DefaultVersion, observedVersion)
	}

	desc, err := c.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		t.Fatal(err)
	}
	var sa = desc.WorkflowExecutionInfo.SearchAttributes
	if sa != nil {
		if val, ok := sa.IndexedFields["TemporalChangeVersion"]; ok {
			t.Errorf("TemporalChangeVersion was set to %s, but the patch ran during replay and should not have written a marker",
				string(val.Data))
		}
	}
}

func sendUpdate(t *testing.T, c client.Client, workflowID, updateID string) {
	t.Helper()
	handle, err := c.UpdateWorkflow(t.Context(), client.UpdateWorkflowOptions{
		UpdateID:     updateID,
		WorkflowID:   workflowID,
		UpdateName:   "Process",
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		t.Fatalf("UpdateWorkflow %s: %v", updateID, err)
	}
	if err := handle.Get(t.Context(), nil); err != nil {
		t.Fatalf("update %s: %v", updateID, err)
	}
}

func captureHistoryToFile(t *testing.T, c client.Client, workflowID, path string) []*history.HistoryEvent {
	t.Helper()
	var events []*history.HistoryEvent
	var raw []json.RawMessage
	iter := c.GetWorkflowHistory(t.Context(), workflowID, "", false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		ev, err := iter.Next()
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, ev)
		b, err := protojson.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, b)
	}
	out, err := json.MarshalIndent(map[string]any{"events": raw}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return events
}
