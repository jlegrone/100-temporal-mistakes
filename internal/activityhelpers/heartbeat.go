package activityhelpers

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
)

// defaultHeartbeatFrequency is used when activity.Info does not report a
// HeartbeatTimeout. Without a configured timeout there is no deadline to
// bisect, so a sensible default frequency is applied directly.
const defaultHeartbeatFrequency = 30 * time.Second

// AutoHeartbeat starts a background goroutine that records activity heartbeats
// at half of the activity's configured HeartbeatTimeout, falling back to a
// 30s frequency if none is set. The returned [context.CancelFunc] stops the
// heartbeat goroutine and should be called when the activity completes; the
// goroutine also exits on its own when ctx is canceled.
//
// AutoHeartbeat is a no-op when ctx is not an activity context; a no-op cancel
// function is returned.
func AutoHeartbeat(ctx context.Context) context.CancelFunc {
	if !activity.IsActivity(ctx) {
		return func() {}
	}

	frequency := activity.GetInfo(ctx).HeartbeatTimeout / 2
	if frequency <= 0 {
		frequency = defaultHeartbeatFrequency
	}

	heartbeatCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(frequency)
		defer ticker.Stop()
		for {
			activity.RecordHeartbeat(ctx)
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return cancel
}
