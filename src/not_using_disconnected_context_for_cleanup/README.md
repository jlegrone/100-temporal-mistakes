# Not Using a Disconnected Context for Cleanup

> [!TIP]
> Activities or [child workflows](../terms/child-workflow.md) started with a canceled context are never dispatched. Use `workflow.NewDisconnectedContext()` for any cleanup that must run after [cancelation](../terms/cancelation.md).

When a workflow is canceled, the root context and all descendants are canceled. If cleanup code (compensation, resource release, notifications) uses the original context, it silently fails -- the activity is never scheduled and `Get()` returns `CanceledError` immediately.

<!--SNIPSTART not-using-disconnected-context-bad-->
[not_using_disconnected_context_for_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_disconnected_context_for_cleanup/workflow.go)
```go

// PurchaseItemV1 blocks on shipping with no fulfilment deadline and no
// compensating action. If the workflow is canceled or shipping hangs, the
// customer is left charged for an item that never arrives.
func PurchaseItemV1(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Hour,
	})

	var ship ShipItemResponse
	if err := workflow.ExecuteActivity(ctx, ShipItem, ShipItemRequest{OrderID: req.OrderID}).Get(ctx, &ship); err != nil {
		return nil, err
	}
	return &PurchaseItemResponse{TrackingID: ship.TrackingID}, nil
}

```
<!--SNIPEND-->

Use a disconnected context for cleanup, so the activity runs even after the workflow is canceled:

<!--SNIPSTART not-using-disconnected-context-good-->
[not_using_disconnected_context_for_cleanup/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_disconnected_context_for_cleanup/workflow.go)
```go

// PurchaseItemV2 fans in shipping completion, a 5-minute fulfilment deadline,
// and workflow cancelation through a Selector. If shipping doesn't win, it
// refunds via an abandoned child workflow started on a disconnected context,
// so the cleanup survives the parent's cancelation.
func PurchaseItemV2(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Hour,
	})

	shipFuture := workflow.ExecuteActivity(ctx, ShipItem, ShipItemRequest{OrderID: req.OrderID})

	var (
		shipResp ShipItemResponse
		err      error
		sel      = workflow.NewNamedSelector(ctx, "shipment")
	)
	sel.AddFuture(shipFuture, func(f workflow.Future) {
		err = f.Get(ctx, &shipResp)
	})
	sel.AddFuture(workflow.NewTimer(ctx, 12*time.Hour), func(f workflow.Future) {
		err = workflow.ErrDeadlineExceeded
	})
	sel.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) {
		err = ctx.Err()
	})
	sel.Select(ctx) // fires when the item is shipped, the timer fires, or ctx is canceled

	if err != nil {
		if startErr := workflowhelpers.StartDisconnectedChildWorkflow(
			ctx,
			RefundPayment,
			RefundPaymentRequest{OrderID: req.OrderID, CustomerID: req.CustomerID},
			workflow.ChildWorkflowOptions{WorkflowID: "refund-" + req.OrderID},
		); startErr != nil {
			return nil, startErr
		}
	}

	return &PurchaseItemResponse{TrackingID: shipResp.TrackingID}, err
}

```
<!--SNIPEND-->

See also: [Deadlocking When a Workflow Is Canceled](../deadlocking_when_workflow_canceled/).
