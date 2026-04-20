# Passing Too Much Information from Activities

> [!TIP]
> Activity results are persisted in [workflow history](terms/event-history.md) -- every byte counts toward history size limits and [replay](terms/replay.md) performance. Return only what the workflow actually needs, and store large data externally.

Activities often fetch or produce data -- database records, API responses, file contents -- and a common mistake is returning all of it when the workflow only needs a small subset. Every activity result is serialized and stored as an event in the workflow history, then replayed in full when a workflow resumes. Oversized results push the workflow closer to the [history length limit](overflowing-workflow-history-length.md), slow down replay, risk hitting the [individual payload size limit](overflowing-maximum-individual-payload-size.md), and increase storage costs across millions of executions.

Design activity return types the same way you'd design an API response -- include only the fields the caller needs:

```go
// Before: returning the full record
func LookupCustomer(ctx context.Context, customerID string) (*Customer, error) {
    return db.GetCustomer(ctx, customerID)
}

// After: returning only what the workflow needs
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

If a downstream activity genuinely needs the full data, have it fetch the data directly rather than passing it through the workflow. For large data that must flow through the system, store it externally (S3, a database, a cache) and pass only a reference (a URL, an ID, a key) as the activity result.

See also: [Overflowing maximum individual payload size](overflowing-maximum-individual-payload-size.md), [Overflowing workflow history bytes](overflowing-workflow-history-bytes.md).
