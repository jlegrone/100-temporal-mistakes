# Not Using Activity Heartbeat Details

> [!TIP]
> Activity [heartbeats](terms/heartbeat.md) can carry progress information, not just keep-alive signals. When a long-running activity is retried, the new attempt can retrieve the last heartbeat details and resume from where it left off.

Most developers who heartbeat understand the keep-alive purpose but miss that heartbeats can carry arbitrary data. Without heartbeat details, a retried activity that processed 9,500 of 10,000 records starts over from record 1. With heartbeat details, it picks up at record 9,501.

<!--SNIPSTART not-using-heartbeat-details-->
[not_using_activity_heartbeat_details/activity.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_activity_heartbeat_details/activity.go)
```go

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

```
<!--SNIPEND-->

Keep heartbeat details small (an index or small checkpoint struct, not your entire activity state). Set a [heartbeat timeout](terms/heartbeat-timeout.md) on the activity -- without one, the server doesn't monitor heartbeats and won't fail the activity if the [worker](terms/worker.md) crashes, which means no retry and no resumption.
