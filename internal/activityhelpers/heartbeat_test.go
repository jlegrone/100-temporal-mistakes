package activityhelpers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
)

func TestAutoHeartbeat_NonActivityContext(t *testing.T) {
	assert.NotPanics(t, func() {
		cancel := AutoHeartbeat(t.Context())
		assert.NotNil(t, cancel)
		cancel()
	})
}

func TestAutoHeartbeat_RecordsHeartbeatFromActivity(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()

	var calls atomic.Int32
	env.SetOnActivityHeartbeatListener(func(_ *activity.Info, _ converter.EncodedValues) {
		calls.Add(1)
	})

	activityFn := func(ctx context.Context) error {
		cancel := AutoHeartbeat(ctx)
		defer cancel()
		// Give the goroutine a moment to record its first heartbeat before
		// the activity returns.
		time.Sleep(50 * time.Millisecond)
		return nil
	}
	env.RegisterActivity(activityFn)
	_, err := env.ExecuteActivity(activityFn)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls.Load(), int32(1))
}
