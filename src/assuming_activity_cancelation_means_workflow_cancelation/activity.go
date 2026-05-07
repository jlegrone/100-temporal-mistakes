package assuming_activity_cancelation_means_workflow_cancelation

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/sdk/activity"
)

// @@@SNIPSTART assuming-activity-cancelation-means-workflow-cancelation-bad

// ProcessOrderActivityBad assumes cancelation means the workflow is done.
func ProcessOrderActivityBad(ctx context.Context, orderID string) error {
	if err := startProcessing(ctx, orderID); err != nil {
		return err
	}

	// Poll for completion, checking for cancelation
	for !isComplete(ctx, orderID) {
		activity.RecordHeartbeat(ctx, orderID)

		// Wrong: assumes the whole workflow is canceled,
		// so tries to clean up directly. But the workflow
		// may have just timed out this activity and wants
		// to retry or take a different path.
		if err := ctx.Err(); err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				db.UpdateStatus(ctx, orderID, "canceled")
			}
			return err
		}

		time.Sleep(time.Second)
	}

	return nil
}

// @@@SNIPEND

// db is a stub for database operations.
var db = &dbStub{}

type dbStub struct{}

func (d *dbStub) UpdateStatus(_ context.Context, _, _ string) error { return nil }

// startProcessing is a stub.
func startProcessing(_ context.Context, _ string) error { return nil }

// isComplete is a stub.
func isComplete(_ context.Context, _ string) bool { return true }
