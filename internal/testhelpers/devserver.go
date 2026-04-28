package testhelpers

import (
	"fmt"
	"testing"

	"go.temporal.io/sdk/client"
	sdktestsuite "go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

// StartDevServerWorker starts a DevServer, creates a worker with a unique task queue,
// calls register to let the caller register workflows and activities, starts the worker,
// and registers cleanup for all resources via t.Cleanup.
func StartDevServerWorker(t testing.TB, register func(r worker.Registry), extraArgs ...string) (c client.Client, taskQueue string) {
	t.Helper()

	server, err := sdktestsuite.StartDevServer(t.Context(), sdktestsuite.DevServerOptions{
		LogLevel:  "error",
		ExtraArgs: extraArgs,
	})
	if err != nil {
		t.Fatalf("StartDevServer: %v", err)
	}
	t.Cleanup(func() { server.Stop() })

	c = server.Client()
	t.Cleanup(func() { c.Close() })

	taskQueue = fmt.Sprintf("%s-%s", t.Name(), t.TempDir())
	w := worker.New(c, taskQueue, worker.Options{})
	register(w)
	if err := w.Start(); err != nil {
		t.Fatalf("worker.Start: %v", err)
	}
	t.Cleanup(func() { w.Stop() })

	return c, taskQueue
}
