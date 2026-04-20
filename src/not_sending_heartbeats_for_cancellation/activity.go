package not_sending_heartbeats_for_cancellation

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// Input contains items to process.
type Input struct {
	Items []string
}

func processItem(item string) {
	fmt.Printf("Processing item: %s\n", item)
}

// @@@SNIPSTART not-sending-heartbeats-for-cancellation

func LongRunningActivity(ctx context.Context, input Input) error {
	for i, item := range input.Items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		processItem(item)
		activity.RecordHeartbeat(ctx, i) // Cancellation is detected here
	}
	return nil
}

// @@@SNIPEND
