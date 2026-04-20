package assuming_signal_update_order

import (
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"pgregory.net/rapid"
)

// @@@SNIPSTART assuming-signal-update-order-test
func TestWorkflowOrderInvariance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random set of changes with increasing ULIDs.
		n := rapid.IntRange(1, 20).Draw(t, "n")
		changes := make([]Change, n)
		for i := range changes {
			changes[i] = Change{
				ID:   ulid.Make().String(),
				Data: rapid.String().Draw(t, fmt.Sprintf("data[%d]", i)),
			}
		}

		// Shuffle the delivery order.
		shuffled := rapid.Permutation(changes).Draw(t, "order")

		// Run the workflow with updates delivered in random order.
		suite := testsuite.WorkflowTestSuite{}
		env := suite.NewTestWorkflowEnvironment()
		env.RegisterWorkflow(MyWorkflow)

		// Count how many updates are accepted vs rejected.
		var accepted []Change
		env.RegisterDelayedCallback(func() {
			for _, c := range shuffled {
				env.UpdateWorkflow("apply-change", "id-"+c.ID, &testUpdateCallback{
					onAccept: func() { accepted = append(accepted, c) },
				}, c)
			}
			env.SignalWorkflow("done", nil)
		}, 0)

		env.ExecuteWorkflow(MyWorkflow)
		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())

		// Accepted changes must be in strictly increasing ULID order.
		for i := 1; i < len(accepted); i++ {
			require.Greater(t, accepted[i].ID, accepted[i-1].ID,
				"accepted changes must be in strictly increasing ULID order")
		}
	})
}

// @@@SNIPEND

type testUpdateCallback struct {
	onAccept func()
}

func (c *testUpdateCallback) Accept()          {}
func (c *testUpdateCallback) Reject(err error) {}
func (c *testUpdateCallback) Complete(success interface{}, err error) {
	if err == nil && c.onAccept != nil {
		c.onAccept()
	}
}
