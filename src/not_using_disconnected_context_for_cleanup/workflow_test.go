package not_using_disconnected_context_for_cleanup

import (
	"testing"
	"time"

	internaltestsuite "github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// @@@SNIPSTART not-using-disconnected-context-test

func TestV1_NoCompensation(t *testing.T) {
	env := internaltestsuite.NewTestWorkflowEnvironment(t)
	// Shipping never completes within the test window so the cancel interrupts it.
	env.OnActivity(ShipItem, mock.Anything, mock.Anything).After(time.Hour).Return(&ShipItemResponse{}, nil)
	env.OnActivity(Refund, mock.Anything, mock.Anything).Return(&RefundResponse{}, nil).Maybe()

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(PurchaseItemV1, PurchaseItemRequest{OrderID: "o-1"})
	require.True(t, env.IsWorkflowCompleted())
	require.True(t, temporal.IsCanceledError(env.GetWorkflowError()))

	// V1 has no compensation -- the customer is left charged.
	env.AssertActivityNotCalled(t, "Refund", mock.Anything, mock.Anything)
}

func TestV2_RefundsOnCancelation(t *testing.T) {
	env := internaltestsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(RefundPayment)
	env.OnActivity(ShipItem, mock.Anything, mock.Anything).After(time.Hour).Return(&ShipItemResponse{}, nil)
	env.OnActivity(Refund, mock.Anything, mock.Anything).Return(&RefundResponse{}, nil)

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(PurchaseItemV2, PurchaseItemRequest{OrderID: "o-2"})
	require.True(t, env.IsWorkflowCompleted())

	// V2 refunds the customer via the disconnected child workflow.
	env.AssertActivityCalled(t, "Refund", mock.Anything, mock.Anything)
}

func TestV2_RefundsOnFulfilmentTimeout(t *testing.T) {
	env := internaltestsuite.NewTestWorkflowEnvironment(t)
	env.RegisterWorkflow(RefundPayment)
	// Shipping outlasts the workflow's fulfilment deadline.
	env.OnActivity(ShipItem, mock.Anything, mock.Anything).After(48*time.Hour).Return(&ShipItemResponse{}, nil)
	env.OnActivity(Refund, mock.Anything, mock.Anything).Return(&RefundResponse{}, nil)

	env.ExecuteWorkflow(PurchaseItemV2, PurchaseItemRequest{OrderID: "o-3"})
	require.True(t, env.IsWorkflowCompleted())

	env.AssertActivityCalled(t, "Refund", mock.Anything, mock.Anything)
}

// @@@SNIPEND
