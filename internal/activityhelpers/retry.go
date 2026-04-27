package activityhelpers

import (
	"context"
	"math"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const (
	defaultInitialInterval = time.Second
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
	return calculateNewRetryDelay(policy, info.Attempt, policy.BackoffCoefficient)
}

// calculateNewRetryDelay computes a wait time before the next retry of an
// activity at the given attempt number under the supplied retry policy.
//
// A zero policy.BackoffCoefficient is replaced with Temporal's documented
// default of 2.0. The effective coefficient is then the larger of that value
// and the supplied defaultBackoffCoefficient. The result is capped at the
// policy's MaximumInterval when one is set; otherwise no cap is applied beyond
// the overflow guard.
func calculateNewRetryDelay(policy *temporal.RetryPolicy, attempt int32, defaultBackoffCoefficient float64) time.Duration {
	initial := policy.InitialInterval
	if initial <= 0 {
		initial = defaultInitialInterval
	}
	policyCoeff := policy.BackoffCoefficient
	if policyCoeff == 0 {
		policyCoeff = 2.0
	}
	coeff := math.Max(defaultBackoffCoefficient, policyCoeff)

	delay := time.Duration(float64(initial) * math.Pow(coeff, float64(attempt-1)))
	maximum := policy.MaximumInterval
	if delay <= 0 {
		// math.Pow overflowed. Cap to the policy maximum if set, otherwise
		// use a large value.
		if maximum > 0 {
			return maximum
		}
		return time.Hour
	}
	if maximum > 0 && delay > maximum {
		return maximum
	}
	return delay
}
