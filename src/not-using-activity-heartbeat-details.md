# Not Using Activity Heartbeat Details

> [!TIP]
> * Activity [heartbeats](terms/heartbeat.md) can carry progress information, not just keep-alive signals.
> * When a long-running activity is retried, the new attempt can retrieve the last heartbeat details and resume from where the previous attempt left off.
> * This is especially valuable for batch processing, file uploads, or any activity with meaningful progress state.

## What?

Most developers who use activity heartbeats understand their primary purpose: telling the Temporal server that a long-running activity is still alive and hasn't stalled. What many miss is that heartbeats can carry arbitrary data -- progress details that are persisted by the server and made available to the next attempt if the activity needs to be retried.

Without heartbeat details, a retried activity starts from scratch. With heartbeat details, a retried activity can pick up where the last attempt left off.

## Why?

Consider an activity that processes 10,000 records from a database. Without heartbeat details, if the [worker](terms/worker.md) crashes after processing 9,500 records, the next attempt starts over from record 1. That is 9,500 records processed twice for no reason.

The same applies to file uploads (re-uploading from byte 0), data migrations (re-migrating already-migrated rows), or any operation where partial progress is meaningful.

The waste compounds:
- **Time**: Repeating already-completed work adds latency to the overall workflow.
- **Resources**: Redundant API calls, database queries, or compute cycles cost real money.
- **Side effects**: If the work is not fully [idempotent](terms/idempotency.md), re-processing can cause issues. Even if it is idempotent, it is still unnecessary load on downstream systems.

## How?

The pattern is straightforward. During execution, periodically heartbeat with progress information. At the start of each attempt, check for previous heartbeat details and resume from there.

Here is an example in Go for an activity that processes records in batches:

```go
func ProcessRecordsActivity(ctx context.Context, input ProcessRecordsInput) error {
    // Check if there are heartbeat details from a previous attempt
    startIndex := 0
    if activity.HasHeartbeatDetails(ctx) {
        var lastProcessedIndex int
        if err := activity.GetHeartbeatDetails(ctx, &lastProcessedIndex); err == nil {
            startIndex = lastProcessedIndex + 1
        }
    }

    for i := startIndex; i < len(input.RecordIDs); i++ {
        // Process the record
        if err := processRecord(input.RecordIDs[i]); err != nil {
            return err
        }

        // Heartbeat with progress
        activity.RecordHeartbeat(ctx, i)
    }

    return nil
}
```

A few practical notes:

**Don't heartbeat too frequently**: Heartbeats generate network traffic to the Temporal server. The SDK throttles heartbeats by default (typically to 80% of the [heartbeat timeout](terms/heartbeat-timeout.md) interval), but it is still good practice to heartbeat at reasonable intervals (e.g. every N records or every few seconds) rather than after every single item.

**Keep heartbeat details small**: Heartbeat details are serialized and sent over the network. A simple integer index or a small checkpoint struct is fine. Don't stuff the entire state of your activity into the heartbeat.

**Set a heartbeat timeout**: Heartbeat details are only useful if you have a heartbeat timeout configured on the activity. Without a timeout, the server doesn't monitor heartbeats and won't fail the activity if the worker crashes -- which means no retry and no resumption from heartbeat details.
