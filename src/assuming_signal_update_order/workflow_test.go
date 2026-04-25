package assuming_signal_update_order

import (
	"testing"

	"github.com/jlegrone/100-temporal-mistakes/internal/testsuite"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// @@@SNIPSTART assuming-signal-update-order-test
func TestWorkflowOrderInvariance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		env := testsuite.NewTestWorkflowEnvironment(t)

		// Generate changes with increasing ULIDs.
		var changes []Change
		for range rapid.IntRange(0, 19).Draw(t, "n") {
			changes = append(changes, Change{
				ID:   ulid.Make().String(),
				Data: rapid.String().Draw(t, "data"), // TODO: No need for random data here, just Sprintf "data_" plus the index
			})
		}
		// The last change has known data we can assert on.
		changes = append(changes, Change{
			ID:   ulid.Make().String(),
			Data: "last_datum",
		})
		// Shuffle and deliver updates in random order.
		shuffled := rapid.Permutation(changes).Draw(t, "order")

		env.RegisterDelayedCallback(func() {
			for _, c := range shuffled {
				env.UpdateWorkflow("apply-change", "id-"+c.ID, &noopUpdateCallback{}, c)
			}
			env.SignalWorkflow("done", nil)
		}, 0)
		// Swap with MyWorkflowV1 to observe test failure
		env.ExecuteWorkflow(MyWorkflowV2)

		require.NoError(t, env.GetWorkflowError())
		// The workflow always accepts the last-generated change (highest ULID),
		// regardless of delivery order.
		var result string
		require.NoError(t, env.GetWorkflowResult(&result))
		require.Equal(t, "last_datum", result)
	})
}

// @@@SNIPEND

type noopUpdateCallback struct{}

func (c *noopUpdateCallback) Accept()                                 {}
func (c *noopUpdateCallback) Reject(err error)                        {}
func (c *noopUpdateCallback) Complete(success interface{}, err error) {}
