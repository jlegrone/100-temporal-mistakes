// Package testhelpers
//
// activity_policy_interceptor.go is the Go reference implementation of the
// Activity Policy Interceptor specification.
//
// See specs/activitypolicyinterceptor/README.md for the language-agnostic spec.
// Requirement numbers cited in inline comments refer to that document.

package testhelpers

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/activityhelpers"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Policy identifiers (must match conformance_tests.json).
const (
	PolicyScheduleToCloseRequired           = "schedule_to_close_required"
	PolicyMaxAttemptsMustBeZeroOrAtLeast3   = "max_attempts_must_be_zero_or_at_least_3"
	PolicyLocalActivityStartToCloseUnder10s = "local_activity_start_to_close_under_10s_required"
	PolicyTimeoutsPermitRetries             = "timeouts_permit_retries"

	// PolicyViolationErrorType is the value of the ApplicationFailure's `type`
	// field for every violation surfaced by this interceptor (spec req 27).
	PolicyViolationErrorType = "PolicyViolationError"
)

// canonicalOrder is the order used when aggregating multiple violations into
// one PolicyViolationError (spec req 21).
var canonicalOrder = []string{
	PolicyScheduleToCloseRequired,
	PolicyMaxAttemptsMustBeZeroOrAtLeast3,
	PolicyLocalActivityStartToCloseUnder10s,
	PolicyTimeoutsPermitRetries,
}

const (
	// defaultHeartbeatTimeout is applied per spec requirement 9 when neither
	// heartbeat nor start-to-close is set on a regular activity.
	defaultHeartbeatTimeout = 30 * time.Second
	// scheduleToCloseRatio is the lower bound used by spec requirement 16.
	scheduleToCloseRatio = 2.1
	// localActivityMaxStartToClose is the upper bound used by spec requirement 19.
	localActivityMaxStartToClose = 10 * time.Second
)

// Severity is the closed enumeration defined by spec requirement 2.
type Severity int

const (
	SeverityIgnore Severity = iota
	SeverityWarn
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityIgnore:
		return "Ignore"
	case SeverityWarn:
		return "Warn"
	case SeverityError:
		return "Error"
	default:
		return fmt.Sprintf("Severity(%d)", int(s))
	}
}

// defaultSeverities are applied per spec requirement 3 to any policy not
// explicitly configured by the caller.
var defaultSeverities = map[string]Severity{
	PolicyScheduleToCloseRequired:           SeverityError,
	PolicyMaxAttemptsMustBeZeroOrAtLeast3:   SeverityError,
	PolicyLocalActivityStartToCloseUnder10s: SeverityError,
	PolicyTimeoutsPermitRetries:             SeverityError,
}

// ActivityPolicyOptions configures the interceptor per spec requirement 1.
//
// The zero value applies the spec defaults: every policy at SeverityError,
// AutoHeartbeat enabled, and the workflow's built-in logger.
type ActivityPolicyOptions struct {
	// Severities overrides individual policies. Unspecified entries fall back
	// to defaultSeverities.
	Severities map[string]Severity
	// AutoHeartbeat controls the activity-side auto-heartbeat helper. Defaults
	// to true per spec requirement 23.
	AutoHeartbeat *bool
	// Logger is the destination for warn-mode log entries. When nil, the
	// workflow's logger is used.
	Logger *slog.Logger
}

func (o ActivityPolicyOptions) severityFor(policy string) Severity {
	if sev, ok := o.Severities[policy]; ok {
		return sev
	}
	return defaultSeverities[policy]
}

func (o ActivityPolicyOptions) autoHeartbeat() bool {
	if o.AutoHeartbeat == nil {
		return true
	}
	return *o.AutoHeartbeat
}

// PolicyViolationDetails is the structured payload placed at details[0] of
// every PolicyViolationError ApplicationFailure (spec req 29).
type PolicyViolationDetails struct {
	Policies     []string `json:"policies"`
	ActivityType string   `json:"activity_type"`
	WorkflowID   string   `json:"workflow_id"`
	RunID        string   `json:"run_id"`
	Explanation  string   `json:"explanation"`
}

// NewActivityPolicyInterceptor returns a worker interceptor implementing the
// Activity Policy Interceptor spec.
func NewActivityPolicyInterceptor(opts ActivityPolicyOptions) interceptor.WorkerInterceptor {
	return &activityPolicyInterceptor{opts: opts}
}

// ----------------------------------------------------------------------
// Worker interceptor
// ----------------------------------------------------------------------

type activityPolicyInterceptor struct {
	interceptor.WorkerInterceptorBase
	opts ActivityPolicyOptions
}

func (a *activityPolicyInterceptor) InterceptActivity(
	ctx context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &activityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		opts:                           a.opts,
	}
}

func (a *activityPolicyInterceptor) InterceptWorkflow(
	ctx workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &workflowInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		opts:                           a.opts,
	}
}

// ----------------------------------------------------------------------
// Workflow inbound (just wraps outbound)
// ----------------------------------------------------------------------

type workflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	opts ActivityPolicyOptions
}

func (w *workflowInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return w.Next.Init(&workflowOutbound{
		WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: outbound},
		opts:                            w.opts,
	})
}

// ----------------------------------------------------------------------
// Workflow outbound (validation + default heartbeat)
// ----------------------------------------------------------------------

type workflowOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	opts ActivityPolicyOptions
}

func (w *workflowOutbound) ExecuteActivity(
	ctx workflow.Context,
	activityType string,
	args ...interface{},
) workflow.Future {
	opts := workflow.GetActivityOptions(ctx)

	// Apply default heartbeat per spec reqs 9-11 BEFORE evaluating
	// timeouts_permit_retries, so the heartbeat-set escape applies.
	if opts.HeartbeatTimeout == 0 && opts.StartToCloseTimeout == 0 {
		opts.HeartbeatTimeout = defaultHeartbeatTimeout
		ctx = workflow.WithActivityOptions(ctx, opts)
	}

	violations := evaluatePolicies(
		opts.ScheduleToCloseTimeout,
		opts.StartToCloseTimeout,
		opts.HeartbeatTimeout,
		opts.RetryPolicy,
		false, /* isLocal */
	)

	if err := w.handleViolations(ctx, activityType, violations); err != nil {
		fut, settable := workflow.NewFuture(ctx)
		settable.SetError(err)
		return fut
	}
	return w.Next.ExecuteActivity(ctx, activityType, args...)
}

func (w *workflowOutbound) ExecuteLocalActivity(
	ctx workflow.Context,
	activityType string,
	args ...interface{},
) workflow.Future {
	opts := workflow.GetLocalActivityOptions(ctx)
	violations := evaluatePolicies(
		opts.ScheduleToCloseTimeout,
		opts.StartToCloseTimeout,
		0, /* heartbeat: local activities don't support heartbeats */
		opts.RetryPolicy,
		true, /* isLocal */
	)
	if err := w.handleViolations(ctx, activityType, violations); err != nil {
		fut, settable := workflow.NewFuture(ctx)
		settable.SetError(err)
		return fut
	}
	return w.Next.ExecuteLocalActivity(ctx, activityType, args...)
}

// handleViolations applies severity logic. Returns a non-nil error when at
// least one violated policy is configured at SeverityError; otherwise emits
// log entries (or no-ops for SeverityIgnore) and returns nil.
func (w *workflowOutbound) handleViolations(
	ctx workflow.Context,
	activityType string,
	violations []violation,
) error {
	if len(violations) == 0 {
		return nil
	}

	var nonIgnored []violation
	hasError := false
	for _, v := range violations {
		sev := w.opts.severityFor(v.policy)
		if sev == SeverityIgnore {
			continue // spec req 4
		}
		nonIgnored = append(nonIgnored, v)
		if sev == SeverityError {
			hasError = true
		}
	}
	if len(nonIgnored) == 0 {
		return nil
	}

	info := workflow.GetInfo(ctx)
	logger := w.opts.Logger
	if logger == nil {
		// Use the workflow's built-in logger by adapting it to slog.
		logger = slog.New(workflowLogHandler{wfLogger: workflow.GetLogger(ctx)})
	}

	// Spec req 5: emit one log entry per non-ignored violation.
	for _, v := range nonIgnored {
		sev := w.opts.severityFor(v.policy)
		mode := "warn"
		if sev == SeverityError {
			mode = "fail"
		}
		attrs := []any{
			slog.String("policy", v.policy),
			slog.String("activity_type", activityType),
			slog.String("workflow_id", info.WorkflowExecution.ID),
			slog.String("run_id", info.WorkflowExecution.RunID),
			slog.String("mode", mode),
		}
		for k, val := range v.extra {
			attrs = append(attrs, slog.Any(k, val))
		}
		logger.Warn("activity policy violation: "+v.policy, attrs...)
	}

	if !hasError {
		return nil
	}
	return buildPolicyViolationError(nonIgnored, activityType, info.WorkflowExecution.ID, info.WorkflowExecution.RunID)
}

// ----------------------------------------------------------------------
// Policy evaluation (pure function — no SDK context required)
// ----------------------------------------------------------------------

type violation struct {
	policy      string
	explanation string
	extra       map[string]any
}

func evaluatePolicies(
	scheduleToClose, startToClose, heartbeat time.Duration,
	retryPolicy *temporal.RetryPolicy,
	isLocal bool,
) []violation {
	var out []violation

	// Policy 1: schedule_to_close_required (regular and local).
	if scheduleToClose <= 0 {
		out = append(out, violation{
			policy:      PolicyScheduleToCloseRequired,
			explanation: "ScheduleToCloseTimeout must be set to a positive value",
		})
	}

	// Policy 4: max_attempts_must_be_zero_or_at_least_3 (regular and local).
	if retryPolicy != nil && retryPolicy.MaximumAttempts > 0 && retryPolicy.MaximumAttempts < 3 {
		out = append(out, violation{
			policy:      PolicyMaxAttemptsMustBeZeroOrAtLeast3,
			explanation: "RetryPolicy.MaximumAttempts must be 0 (unlimited) or >= 3",
			extra:       map[string]any{"max_attempts": retryPolicy.MaximumAttempts},
		})
	}

	// Policy 19: local_activity_start_to_close_under_10s_required.
	if isLocal {
		if startToClose <= 0 || startToClose >= localActivityMaxStartToClose {
			out = append(out, violation{
				policy:      PolicyLocalActivityStartToCloseUnder10s,
				explanation: "local activity StartToCloseTimeout must be set and < 10s",
				extra:       map[string]any{"start_to_close_seconds": startToClose.Seconds()},
			})
		}
	}

	// Policy 16: timeouts_permit_retries — only when heartbeat is unset on a
	// regular activity with both other timeouts set.
	if !isLocal && heartbeat == 0 && startToClose > 0 && scheduleToClose > 0 {
		minRequired := time.Duration(scheduleToCloseRatio * float64(startToClose))
		if scheduleToClose < minRequired {
			out = append(out, violation{
				policy: PolicyTimeoutsPermitRetries,
				explanation: fmt.Sprintf(
					"ScheduleToCloseTimeout must be >= %vx StartToCloseTimeout when HeartbeatTimeout is unset",
					scheduleToCloseRatio,
				),
			})
		}
	}

	return canonicalize(out)
}

func canonicalize(in []violation) []violation {
	rank := make(map[string]int, len(canonicalOrder))
	for i, p := range canonicalOrder {
		rank[p] = i
	}
	sort.SliceStable(in, func(i, j int) bool {
		return rank[in[i].policy] < rank[in[j].policy]
	})
	return in
}

func buildPolicyViolationError(
	violations []violation,
	activityType, workflowID, runID string,
) error {
	policies := make([]string, len(violations))
	explanations := make([]string, len(violations))
	for i, v := range violations {
		policies[i] = v.policy
		explanations[i] = v.explanation
	}
	explanation := strings.Join(explanations, "; ")
	message := fmt.Sprintf("activity policy violation: %s: %s", strings.Join(policies, ","), explanation)
	details := PolicyViolationDetails{
		Policies:     policies,
		ActivityType: activityType,
		WorkflowID:   workflowID,
		RunID:        runID,
		Explanation:  explanation,
	}
	return temporal.NewApplicationErrorWithOptions(
		message,
		PolicyViolationErrorType,
		temporal.ApplicationErrorOptions{
			NonRetryable: true,
			Details:      []interface{}{details},
		},
	)
}

// ----------------------------------------------------------------------
// Activity inbound (auto-heartbeat)
// ----------------------------------------------------------------------

type activityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	opts ActivityPolicyOptions
}

func (a *activityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (interface{}, error) {
	if !a.opts.autoHeartbeat() {
		return a.Next.ExecuteActivity(ctx, in) // spec req 25
	}
	// Reuse the existing helper, which already implements req 23 (half-cadence
	// with 30s fallback) and req 24 (cancel on return).
	cancel := activityhelpers.AutoHeartbeat(ctx)
	defer cancel()
	return a.Next.ExecuteActivity(ctx, in)
}

// ----------------------------------------------------------------------
// slog -> workflow.Logger adapter
// ----------------------------------------------------------------------

// workflowLogHandler bridges slog.Handler to the Temporal workflow logger.
// Only Warn level is used by the interceptor, but the handler honors all
// levels for completeness.
type workflowLogHandler struct {
	wfLogger interface {
		Debug(msg string, keyvals ...interface{})
		Info(msg string, keyvals ...interface{})
		Warn(msg string, keyvals ...interface{})
		Error(msg string, keyvals ...interface{})
	}
	attrs []slog.Attr
	group string
}

func (h workflowLogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h workflowLogHandler) Handle(_ context.Context, r slog.Record) error {
	keyvals := make([]interface{}, 0, 2*(len(h.attrs)+r.NumAttrs()))
	for _, a := range h.attrs {
		keyvals = append(keyvals, a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		keyvals = append(keyvals, a.Key, a.Value.Any())
		return true
	})
	switch {
	case r.Level >= slog.LevelError:
		h.wfLogger.Error(r.Message, keyvals...)
	case r.Level >= slog.LevelWarn:
		h.wfLogger.Warn(r.Message, keyvals...)
	case r.Level >= slog.LevelInfo:
		h.wfLogger.Info(r.Message, keyvals...)
	default:
		h.wfLogger.Debug(r.Message, keyvals...)
	}
	return nil
}

func (h workflowLogHandler) WithAttrs(as []slog.Attr) slog.Handler {
	out := h
	out.attrs = append(append([]slog.Attr{}, h.attrs...), as...)
	return out
}

func (h workflowLogHandler) WithGroup(name string) slog.Handler {
	out := h
	out.group = name
	return out
}
