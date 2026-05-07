package not_using_activity_heartbeat_details

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// ProcessRecordsInput contains the record IDs to process.
type ProcessRecordsInput struct {
	RecordIDs []string
}

func processRecord(id string) error {
	fmt.Printf("Processing record: %s\n", id)
	return nil
}

// @@@SNIPSTART not-using-heartbeat-details

func ProcessRecordsActivity(ctx context.Context, input ProcessRecordsInput) error {
	startIndex := 0
	if activity.HasHeartbeatDetails(ctx) {
		var lastProcessed int
		if err := activity.GetHeartbeatDetails(ctx, &lastProcessed); err == nil {
			startIndex = lastProcessed + 1
		}
	}

	for i := startIndex; i < len(input.RecordIDs); i++ {
		if err := processRecord(input.RecordIDs[i]); err != nil {
			return err
		}
		activity.RecordHeartbeat(ctx, i)
	}
	return nil
}

// @@@SNIPEND
