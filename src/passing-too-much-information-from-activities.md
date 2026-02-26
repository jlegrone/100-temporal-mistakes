# Passing Too Much Information from Activities

> [!TIP]
> * Activity results are persisted in [workflow history](terms/event-history.md) -- every byte counts toward history size limits and [replay](terms/replay.md) performance.
> * Return only what the workflow actually needs: IDs, status codes, small summaries.
> * Store large data externally (database, blob storage) and pass references instead of full [payloads](terms/payload.md).

## What?

Activities often fetch or produce data -- database records, API responses, file contents, computation results. A common mistake is returning all of that data directly as the activity result, even when the workflow only needs a small subset.

For example, an activity that looks up a customer might return the entire customer record (addresses, order history, preferences, profile image metadata) when the workflow only needs the customer ID and subscription tier to make a routing decision.

## Why?

Every activity result is serialized and stored as an event in the workflow history. That history is what gets [replayed](terms/replay.md) when a workflow resumes after a [worker](terms/worker.md) restart or rebalance. Large activity results have compounding effects:

1. **History bloat**: Each oversized result pushes the workflow closer to the [history size limit](overflowing-workflow-history-size.md). Workflows that would otherwise run for weeks may hit the 50k event or size cap prematurely.
2. **Slower replay**: When a workflow needs to be replayed, the entire history is fetched from the server and processed. Larger payloads mean more data transferred over the network and more time spent deserializing.
3. **Payload size limits**: Individual payloads that exceed the gRPC size limit (4MB by default) will be rejected outright, causing the workflow to [stop making progress](overflowing-maximum-individual-payload-size.md).
4. **Storage costs**: All that data lives in your [Temporal server backend](terms/temporal-server-backend.md). Multiply a 500KB activity result by millions of workflow executions and the storage adds up.

## How?

Design activity return types the same way you'd design an API response -- include only the fields the caller needs.

**Before** (returning everything):
```go
func LookupCustomer(ctx context.Context, customerID string) (*Customer, error) {
    // Returns the full customer record: addresses, orders, preferences, ...
    return db.GetCustomer(ctx, customerID)
}
```

**After** (returning what the workflow needs):
```go
type CustomerSummary struct {
    ID               string
    SubscriptionTier string
    IsActive         bool
}

func LookupCustomer(ctx context.Context, customerID string) (*CustomerSummary, error) {
    customer, err := db.GetCustomer(ctx, customerID)
    if err != nil {
        return nil, err
    }
    return &CustomerSummary{
        ID:               customer.ID,
        SubscriptionTier: customer.SubscriptionTier,
        IsActive:         customer.IsActive,
    }, nil
}
```

If a downstream activity genuinely needs the full data, have that activity fetch it directly rather than passing it through the workflow. The workflow is an orchestrator -- it should pass references, not bulk data.

For cases where large data must flow through the system, store it externally (S3, a database, a cache) and pass the reference (a URL, an ID, a key) as the activity result.
