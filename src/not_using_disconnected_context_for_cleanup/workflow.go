package not_using_disconnected_context_for_cleanup

import (
	"context"
	"time"

	"github.com/jlegrone/100-temporal-mistakes/internal/workflowhelpers"
	"go.temporal.io/sdk/workflow"
)

type PurchaseItemRequest struct {
	OrderID    string
	CustomerID string
}

type PurchaseItemResponse struct {
	TrackingID string
}

type ShipItemRequest struct {
	OrderID string
}

type ShipItemResponse struct {
	TrackingID string
}

type RefundPaymentRequest struct {
	OrderID    string
	CustomerID string
}

type RefundPaymentResponse struct{}

type RefundRequest struct {
	OrderID string
}

type RefundResponse struct{}

// ShipItem dispatches the order to the shipping carrier.
func ShipItem(ctx context.Context, req ShipItemRequest) (*ShipItemResponse, error) {
	return &ShipItemResponse{}, nil
}

// Refund credits the customer's payment method.
func Refund(ctx context.Context, req RefundRequest) (*RefundResponse, error) {
	return &RefundResponse{}, nil
}

// RefundPayment is the compensating child workflow started when a purchase
// fails to ship within the fulfilment window or is canceled.
func RefundPayment(ctx workflow.Context, req RefundPaymentRequest) (*RefundPaymentResponse, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: time.Hour,
	})
	if err := workflow.ExecuteActivity(ctx, Refund, RefundRequest{OrderID: req.OrderID}).Get(ctx, nil); err != nil {
		return nil, err
	}
	return &RefundPaymentResponse{}, nil
}

// @@@SNIPSTART not-using-disconnected-context-bad

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

// @@@SNIPEND

// @@@SNIPSTART not-using-disconnected-context-good

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

// @@@SNIPEND
