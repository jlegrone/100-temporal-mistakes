package workflowhelpers

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

// AwaitActivity executes activity and awaits the result.
// It is similar to workflow.ExecuteActivity but preserves type safety.
func AwaitActivity[Req, Resp any](
	ctx workflow.Context,
	activity func(context.Context, Req) (*Resp, error),
	req Req,
) (*Resp, error) {
	fut := workflow.ExecuteActivity(ctx, activity, req)
	var resp Resp
	if err := fut.Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
