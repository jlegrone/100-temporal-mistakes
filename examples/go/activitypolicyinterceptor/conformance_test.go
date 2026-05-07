package activitypolicy

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	spec "github.com/jlegrone/100-temporal-mistakes/specs/activitypolicyinterceptor"
)

// ----------------------------------------------------------------------
// Conformance test runner: loads ../../../specs/activitypolicyinterceptor/
// conformance_tests.json at runtime and exercises every schedule_activity
// case as a Go subtest. execute_activity cases are skipped — they require
// real-time/heartbeat fixtures beyond what testsuite provides cheaply.
// ----------------------------------------------------------------------

type conformanceFile struct {
	SchemaVersion string            `json:"$schema_version"`
	SpecDocument  string            `json:"spec_document"`
	Description   string            `json:"description"`
	Tests         []conformanceCase `json:"tests"`
}

type conformanceCase struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	SpecRequirement   int                 `json:"spec_requirement"`
	InterceptorConfig conformanceConfig   `json:"interceptor_config"`
	Scenario          conformanceScenario `json:"scenario"`
	Expected          conformanceExpected `json:"expected"`
}

type conformanceConfig struct {
	Severities      map[string]string `json:"Severities"`
	AutoHeartbeat   *bool             `json:"AutoHeartbeat"`
	RequiredRetries *int              `json:"RequiredRetries"`
}

type conformanceScenario struct {
	Type            string                  `json:"type"`
	ActivityKind    string                  `json:"activity_kind"`
	ActivityOptions conformanceActivityOpts `json:"activity_options"`
	ActivityRuntime *conformanceRuntime     `json:"activity_runtime"`
}

type conformanceActivityOpts struct {
	ScheduleToCloseSeconds *float64                `json:"schedule_to_close_seconds"`
	StartToCloseSeconds    *float64                `json:"start_to_close_seconds"`
	HeartbeatSeconds       *float64                `json:"heartbeat_seconds"`
	RetryPolicy            *conformanceRetryPolicy `json:"retry_policy"`
}

type conformanceRetryPolicy struct {
	InitialIntervalSeconds *float64 `json:"initial_interval_seconds"`
	BackoffCoefficient     *float64 `json:"backoff_coefficient"`
	MaximumIntervalSeconds *float64 `json:"maximum_interval_seconds"`
	MaximumAttempts        *int32   `json:"maximum_attempts"`
}

type conformanceRuntime struct {
	DurationSeconds  float64 `json:"duration_seconds"`
	ManualHeartbeats int     `json:"manual_heartbeats"`
}

type conformanceExpected struct {
	Outcome                   string                 `json:"outcome"`
	ApplicationFailure        *conformanceAppFailure `json:"application_failure"`
	LoggedPolicies            []string               `json:"logged_policies"`
	MutatedOptions            *conformanceMutated    `json:"mutated_options"`
	ActivityScheduledOnWorker *bool                  `json:"activity_scheduled_on_worker"`
	ViolationLogEmitted       *bool                  `json:"violation_log_emitted"`
}

type conformanceAppFailure struct {
	Type                   string   `json:"type"`
	NonRetryable           bool     `json:"non_retryable"`
	DetailsPayloadPolicies []string `json:"details_payload_policies"`
}

type conformanceMutated struct {
	HeartbeatSeconds *float64 `json:"heartbeat_seconds"`
}

// loadConformanceFile parses the JSON embedded by the spec package. Using
// the embedded copy guarantees the Go test runner uses the exact bytes shipped
// with the spec — no path lookups, no risk of stale on-disk copies.
func loadConformanceFile(t *testing.T) conformanceFile {
	t.Helper()
	require.NotEmpty(t, spec.ConformanceJSON, "embedded conformance JSON is empty")
	var cf conformanceFile
	require.NoError(t, json.Unmarshal(spec.ConformanceJSON, &cf), "parse embedded conformance JSON")
	require.NotEmpty(t, cf.Tests)
	return cf
}

func TestConformance(t *testing.T) {
	cf := loadConformanceFile(t)
	for _, c := range cf.Tests {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			runConformanceCase(t, c)
		})
	}
}

func runConformanceCase(t *testing.T, c conformanceCase) {
	if c.Scenario.Type == "execute_activity" {
		t.Skipf("execute_activity scenarios require activity-runtime fixtures (req %d)", c.SpecRequirement)
		return
	}
	if c.Scenario.Type != "schedule_activity" {
		t.Fatalf("unsupported scenario.type %q", c.Scenario.Type)
	}

	opts, err := buildOptions(c.InterceptorConfig)
	require.NoError(t, err)
	rec := newRecordingHandler()
	opts.Logger = slog.New(rec)

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{New(opts)},
	})

	var (
		activityRan       atomic.Bool
		appliedHeartbeat  atomic.Int64
		appliedHeartbeatN atomic.Int32 // 0 = unset, >0 = duration was set
	)
	captureActivity := func(ctx context.Context) (string, error) {
		activityRan.Store(true)
		info := activity.GetInfo(ctx)
		appliedHeartbeat.Store(int64(info.HeartbeatTimeout))
		appliedHeartbeatN.Add(1)
		return "ok", nil
	}
	env.RegisterActivityWithOptions(captureActivity, activity.RegisterOptions{Name: "noopActivity"})

	wfOpts := buildActivityOptions(c.Scenario.ActivityOptions)
	localOpts := buildLocalActivityOptions(c.Scenario.ActivityOptions)
	isLocal := c.Scenario.ActivityKind == "local"

	wf := func(ctx workflow.Context) error {
		if isLocal {
			ctx = workflow.WithLocalActivityOptions(ctx, localOpts)
			return workflow.ExecuteLocalActivity(ctx, captureActivity).Get(ctx, nil)
		}
		ctx = workflow.WithActivityOptions(ctx, wfOpts)
		return workflow.ExecuteActivity(ctx, captureActivity).Get(ctx, nil)
	}
	env.RegisterWorkflowWithOptions(wf, workflow.RegisterOptions{Name: "conformanceWf-" + c.ID})

	env.ExecuteWorkflow(wf)
	require.True(t, env.IsWorkflowCompleted())
	wfErr := env.GetWorkflowError()

	switch c.Expected.Outcome {
	case "violation":
		require.Error(t, wfErr)
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(wfErr, &appErr),
			"expected ApplicationError, got %T: %v", wfErr, wfErr)
		assert.Equal(t, c.Expected.ApplicationFailure.Type, appErr.Type())
		assert.Equal(t, c.Expected.ApplicationFailure.NonRetryable, appErr.NonRetryable())
		var details ViolationDetails
		require.NoError(t, appErr.Details(&details))
		assert.Equal(t,
			c.Expected.ApplicationFailure.DetailsPayloadPolicies,
			details.Policies,
			"violated policies",
		)
		if c.Expected.ActivityScheduledOnWorker != nil {
			assert.Equal(t, *c.Expected.ActivityScheduledOnWorker, activityRan.Load(),
				"activity_scheduled_on_worker")
		}
	case "forwarded":
		require.NoError(t, wfErr, "expected forwarded; got %v", wfErr)
		assert.True(t, activityRan.Load(), "activity should have run")
		if c.Expected.ViolationLogEmitted != nil && !*c.Expected.ViolationLogEmitted {
			assert.Empty(t, rec.policies(), "no violation log entries expected")
		}
	case "forwarded_with_mutation":
		require.NoError(t, wfErr, "expected forwarded_with_mutation; got %v", wfErr)
		assert.True(t, activityRan.Load(), "activity should have run")
		if c.Expected.MutatedOptions != nil && c.Expected.MutatedOptions.HeartbeatSeconds != nil {
			expected := time.Duration(*c.Expected.MutatedOptions.HeartbeatSeconds * float64(time.Second))
			actual := time.Duration(appliedHeartbeat.Load())
			assert.Equal(t, expected, actual,
				"applied heartbeat_timeout mismatch")
		}
	case "warn_logged_and_forwarded":
		require.NoError(t, wfErr, "expected warn_logged_and_forwarded; got %v", wfErr)
		assert.True(t, activityRan.Load(), "activity should have run")
		assert.ElementsMatch(t, c.Expected.LoggedPolicies, rec.policies(),
			"logged policies mismatch")
	default:
		t.Fatalf("unsupported expected.outcome %q", c.Expected.Outcome)
	}
}

// ----------------------------------------------------------------------
// JSON → SDK type conversions
// ----------------------------------------------------------------------

func buildOptions(c conformanceConfig) (Options, error) {
	out := Options{}
	if len(c.Severities) > 0 {
		sev := make(map[string]Severity, len(c.Severities))
		for policy, name := range c.Severities {
			parsed, err := parseSeverity(name)
			if err != nil {
				return Options{}, err
			}
			sev[policy] = parsed
		}
		out.Severities = sev
	}
	if c.AutoHeartbeat != nil {
		v := *c.AutoHeartbeat
		out.AutoHeartbeat = &v
	}
	if c.RequiredRetries != nil {
		v := *c.RequiredRetries
		out.RequiredRetries = &v
	}
	return out, nil
}

func parseSeverity(name string) (Severity, error) {
	switch name {
	case "Ignore":
		return SeverityIgnore, nil
	case "Warn":
		return SeverityWarn, nil
	case "Error":
		return SeverityError, nil
	default:
		return 0, errors.New("unknown severity: " + name)
	}
}

func buildActivityOptions(o conformanceActivityOpts) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		ScheduleToCloseTimeout: secondsToDuration(o.ScheduleToCloseSeconds),
		StartToCloseTimeout:    secondsToDuration(o.StartToCloseSeconds),
		HeartbeatTimeout:       secondsToDuration(o.HeartbeatSeconds),
		RetryPolicy:            buildRetryPolicy(o.RetryPolicy),
	}
}

func buildLocalActivityOptions(o conformanceActivityOpts) workflow.LocalActivityOptions {
	return workflow.LocalActivityOptions{
		ScheduleToCloseTimeout: secondsToDuration(o.ScheduleToCloseSeconds),
		StartToCloseTimeout:    secondsToDuration(o.StartToCloseSeconds),
		RetryPolicy:            buildRetryPolicy(o.RetryPolicy),
	}
}

func buildRetryPolicy(r *conformanceRetryPolicy) *temporal.RetryPolicy {
	if r == nil {
		return nil
	}
	rp := &temporal.RetryPolicy{}
	if r.InitialIntervalSeconds != nil {
		rp.InitialInterval = secondsToDurationVal(*r.InitialIntervalSeconds)
	}
	if r.BackoffCoefficient != nil {
		rp.BackoffCoefficient = *r.BackoffCoefficient
	}
	if r.MaximumIntervalSeconds != nil {
		rp.MaximumInterval = secondsToDurationVal(*r.MaximumIntervalSeconds)
	}
	if r.MaximumAttempts != nil {
		rp.MaximumAttempts = *r.MaximumAttempts
	}
	return rp
}

func secondsToDuration(s *float64) time.Duration {
	if s == nil {
		return 0
	}
	return secondsToDurationVal(*s)
}

func secondsToDurationVal(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

// ----------------------------------------------------------------------
// slog.Handler that captures policy-violation log records by policy name.
// Records the value of the "policy" attribute on every Warn-or-higher entry.
// ----------------------------------------------------------------------

type recordingHandler struct {
	mu      sync.Mutex
	records []string
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{}
}

func (h *recordingHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= slog.LevelWarn
}

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	if !strings.HasPrefix(r.Message, "activity policy violation") {
		return nil
	}
	var policy string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "policy" {
			policy = a.Value.String()
			return false
		}
		return true
	})
	if policy == "" {
		return nil
	}
	h.mu.Lock()
	h.records = append(h.records, policy)
	h.mu.Unlock()
	return nil
}

func (h *recordingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *recordingHandler) policies() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, len(h.records))
	copy(out, h.records)
	return out
}
