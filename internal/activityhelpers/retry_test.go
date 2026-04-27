package activityhelpers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

func TestGetNextRetryDelay(t *testing.T) {
	tests := map[string]struct {
		info      activity.Info
		wantDelay time.Duration
	}{
		"nil retry policy": {
			info:      activity.Info{Attempt: 1, RetryPolicy: nil},
			wantDelay: 0,
		},
		"first attempt, default-ish policy": {
			info: activity.Info{
				Attempt: 1,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    100 * time.Second,
					MaximumAttempts:    0,
				},
			},
			wantDelay: time.Second,
		},
		"second attempt doubles": {
			info: activity.Info{
				Attempt: 2,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    100 * time.Second,
				},
			},
			wantDelay: 2 * time.Second,
		},
		"third attempt quadruples": {
			info: activity.Info{
				Attempt: 3,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    100 * time.Second,
				},
			},
			wantDelay: 4 * time.Second,
		},
		"capped at maximum interval": {
			info: activity.Info{
				Attempt: 10,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    30 * time.Second,
				},
			},
			wantDelay: 30 * time.Second,
		},
		"maximum attempts reached": {
			info: activity.Info{
				Attempt: 5,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumAttempts:    5,
				},
			},
			wantDelay: 0,
		},
		"final allowed attempt computes next delay": {
			info: activity.Info{
				Attempt: 4,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    100 * time.Second,
					MaximumAttempts:    5,
				},
			},
			wantDelay: 8 * time.Second,
		},
		"unlimited maximum attempts": {
			info: activity.Info{
				Attempt: 100,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    60 * time.Second,
					MaximumAttempts:    0,
				},
			},
			wantDelay: 60 * time.Second,
		},
		"backoff coefficient of 1 keeps interval constant": {
			info: activity.Info{
				Attempt: 7,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    250 * time.Millisecond,
					BackoffCoefficient: 1.0,
					MaximumInterval:    10 * time.Second,
				},
			},
			wantDelay: 250 * time.Millisecond,
		},
		"zero initial interval defaults to one second": {
			info: activity.Info{
				Attempt: 1,
				RetryPolicy: &temporal.RetryPolicy{
					BackoffCoefficient: 2.0,
				},
			},
			wantDelay: time.Second,
		},
		"zero backoff coefficient defaults to two": {
			info: activity.Info{
				Attempt: 3,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval: time.Second,
				},
			},
			wantDelay: 4 * time.Second,
		},
		"zero maximum interval defaults to one hundred times initial": {
			info: activity.Info{
				Attempt: 20,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
				},
			},
			wantDelay: 100 * time.Second,
		},
		"large attempt does not overflow": {
			info: activity.Info{
				Attempt: 1000,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 2.0,
					MaximumInterval:    5 * time.Minute,
				},
			},
			wantDelay: 5 * time.Minute,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.wantDelay, getNextRetryDelay(tc.info))
		})
	}
}

// TestGetNextRetryDelay_NonActivityContext verifies that the exported function
// is safe to call with a plain context (e.g. one returned by t.Context()) and
// does not panic when invoked outside of a Temporal activity.
func TestGetNextRetryDelay_NonActivityContext(t *testing.T) {
	assert.NotPanics(t, func() {
		assert.Equal(t, time.Duration(0), GetNextRetryDelay(t.Context()))
	})
}
