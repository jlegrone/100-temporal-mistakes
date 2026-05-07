package testhelpers

import (
	"fmt"
	"testing"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	sdktestsuite "go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/jlegrone/100-temporal-mistakes/examples/go/activitypolicyinterceptor"
)

// StartDevServerWorker starts a DevServer, creates a worker with a unique
// task queue, calls register to let the caller register workflows and
// activities, starts the worker, and registers cleanup for all resources via
// t.Cleanup.
//
// The Activity Policy Interceptor is installed on the worker at its strictest
// defaults; pass [WithActivityPolicySeverity] to lower or silence individual
// policies for tests demonstrating non-compliant patterns. Use
// [WithDevServerExtraArgs] to forward extra command-line args to the dev
// server.
func StartDevServerWorker(t testing.TB, register func(r worker.Registry), opts ...HelperOption) (c client.Client, taskQueue string) {
	t.Helper()

	cfg := newHelperConfig(opts)

	server, err := sdktestsuite.StartDevServer(t.Context(), sdktestsuite.DevServerOptions{
		LogLevel:  "error",
		ExtraArgs: cfg.devServerExtraArgs,
	})
	if err != nil {
		t.Fatalf("StartDevServer: %v", err)
	}
	t.Cleanup(func() { server.Stop() })

	c = server.Client()
	t.Cleanup(func() { c.Close() })

	taskQueue = fmt.Sprintf("%s-%s", t.Name(), t.TempDir())
	w := worker.New(c, taskQueue, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{
			activitypolicy.New(cfg.activityPolicyOptions()),
		},
	})
	register(w)
	if err := w.Start(); err != nil {
		t.Fatalf("worker.Start: %v", err)
	}
	t.Cleanup(func() { w.Stop() })

	return c, taskQueue
}
