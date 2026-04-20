package not_validating_replay_safety_before_deployments

import (
	"go.temporal.io/sdk/workflow"
)

// MyWorkflow is a stub workflow used in replay testing examples.
func MyWorkflow(_ workflow.Context) error {
	return nil
}
