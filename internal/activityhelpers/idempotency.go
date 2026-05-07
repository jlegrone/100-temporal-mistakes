package activityhelpers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"go.temporal.io/sdk/activity"
)

// GetIdempotencyToken returns a deterministic idempotency key for the
// currently executing activity, derived from the SHA-256 of the workflow ID,
// workflow run ID, and activity ID. Retries of the same activity produce the
// same token, making it suitable to pass to external services that accept an
// idempotency key.
//
// Returns the empty string if ctx is not an activity context.
func GetIdempotencyToken(ctx context.Context) string {
	if !activity.IsActivity(ctx) {
		return ""
	}
	return getIdempotencyToken(activity.GetInfo(ctx))
}

func getIdempotencyToken(info activity.Info) string {
	h := sha256.New()
	// Null bytes act as a delimiter so that, e.g., workflow ID "foo" with
	// activity ID "bar" cannot collide with workflow ID "foob" and activity ID
	// "ar". Temporal IDs are UTF-8 strings that do not contain null bytes.
	h.Write([]byte(info.WorkflowExecution.ID))
	h.Write([]byte{0})
	h.Write([]byte(info.WorkflowExecution.RunID))
	h.Write([]byte{0})
	h.Write([]byte(info.ActivityID))
	return hex.EncodeToString(h.Sum(nil))
}
