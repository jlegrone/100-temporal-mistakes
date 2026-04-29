package testhelpers

import "github.com/jlegrone/100-temporal-mistakes/examples/go/activitypolicyinterceptor"

// HelperOption configures the testhelpers helper functions
// (ReplayWorkflowHistoryFromJSONFile, AssertWorkflowReplayFromJSONFiles,
// StartDevServerWorker).
//
// NewTestWorkflowEnvironment does not take HelperOptions because tests can
// override worker options on the returned env directly via
// env.SetWorkerOptions.
type HelperOption func(*helperConfig)

type helperConfig struct {
	policySeverities   map[string]activitypolicy.Severity
	devServerExtraArgs []string
}

func newHelperConfig(opts []HelperOption) helperConfig {
	var c helperConfig
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// WithActivityPolicySeverity overrides the severity of a single Activity
// Policy Interceptor policy for this helper invocation. Multiple calls layer
// on top of each other, with later calls overriding earlier ones for the same
// policy. Policies not specified keep their default Error severity.
//
// Use this in tests that exist specifically to demonstrate the issues the
// policy interceptor catches: lower the severity of the violated policy to
// activitypolicy.SeverityWarn to keep the warn-mode log output (proving the
// test is exercising a real mistake) without blocking the activity from being
// scheduled, or to activitypolicy.SeverityIgnore to silence the policy
// entirely.
func WithActivityPolicySeverity(policy string, severity activitypolicy.Severity) HelperOption {
	return func(c *helperConfig) {
		if c.policySeverities == nil {
			c.policySeverities = map[string]activitypolicy.Severity{}
		}
		c.policySeverities[policy] = severity
	}
}

// WithDevServerExtraArgs forwards extra command-line args to the embedded
// Temporal dev server (e.g. "--enable-time-skipping").
func WithDevServerExtraArgs(args ...string) HelperOption {
	return func(c *helperConfig) { c.devServerExtraArgs = args }
}

// activityPolicyOptions builds an Options value from the helper config. The
// strictest defaults apply to any policy not overridden.
func (c helperConfig) activityPolicyOptions() activitypolicy.Options {
	return activitypolicy.Options{Severities: c.policySeverities}
}
