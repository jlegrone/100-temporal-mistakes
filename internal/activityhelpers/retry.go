package activityhelpers

import (
	"context"
	"math"
	"time"

	"go.temporal.io/sdk/activity"
)

const (
	defaultInitialInterval    = time.Second
	defaultBackoffCoefficient = 2.0
	defaultMaximumIntervalMul = 100
)

// GetNextRetryDelay reports how long the Temporal service will wait before
// scheduling the next attempt of this activity if the current attempt fails.
// Returns 0 if ctx is not an activity context.
//
// The calculation mirrors Temporal's server-side retry algorithm:
//
//	delay = min(InitialInterval * BackoffCoefficient^(Attempt-1), MaximumInterval)
//
// Defaults are applied when the policy fields are zero, matching the documented
// behavior of [go.temporal.io/sdk/temporal.RetryPolicy]. The result does not
// account for the activity's ScheduleToClose deadline; the server may shorten
// the wait if the deadline is closer than the computed delay.
func GetNextRetryDelay(ctx context.Context) time.Duration {
	if !activity.IsActivity(ctx) {
		return 0
	}
	return getNextRetryDelay(activity.GetInfo(ctx))
}

func getNextRetryDelay(info activity.Info) time.Duration {
	policy := info.RetryPolicy
	if policy == nil {
		return 0
	}
	if policy.MaximumAttempts > 0 && info.Attempt >= policy.MaximumAttempts {
		return 0
	}

	initial := policy.InitialInterval
	if initial <= 0 {
		initial = defaultInitialInterval
	}
	coeff := policy.BackoffCoefficient
	if coeff < 1 {
		coeff = defaultBackoffCoefficient
	}
	maximum := policy.MaximumInterval
	if maximum <= 0 {
		maximum = defaultMaximumIntervalMul * initial
	}

	delay := time.Duration(float64(initial) * math.Pow(coeff, float64(info.Attempt-1)))
	if delay <= 0 || delay > maximum {
		delay = maximum
	}
	return delay
}
