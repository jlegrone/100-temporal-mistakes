package workflowhelpers

import (
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

// StartDisconnectedChildWorkflow starts a child workflow on a context that
// is disconnected from the parent's cancelation, with ParentClosePolicy set
// to ABANDON so the child runs to completion even if the parent is canceled
// or terminated.
//
// It blocks until the server has accepted the start command (so the child is
// durably scheduled), then returns; the child's result is not awaited.
//
// Useful for compensating actions that must run even when the parent
// workflow has been canceled, such as refunding a payment after a fulfilment
// deadline has been exceeded.
func StartDisconnectedChildWorkflow[Req, Resp any](
	ctx workflow.Context,
	childWorkflow func(workflow.Context, Req) (*Resp, error),
	req Req,
	opts workflow.ChildWorkflowOptions,
) error {
	opts.ParentClosePolicy = enums.PARENT_CLOSE_POLICY_ABANDON
	disconnectedCtx, _ := workflow.NewDisconnectedContext(workflow.WithChildOptions(ctx, opts))
	fut := workflow.ExecuteChildWorkflow(disconnectedCtx, childWorkflow, req)
	return fut.GetChildWorkflowExecution().Get(disconnectedCtx, nil)
}
