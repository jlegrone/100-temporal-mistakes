package not_using_temporal_sdk_for_observability

import (
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART not-using-temporal-sdk-for-observability-good

// Good: use the SDK's replay-aware workflow logger.
func MyWorkflow(ctx workflow.Context, orderID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Processing order", "orderID", orderID)
	return nil
}

// @@@SNIPEND
