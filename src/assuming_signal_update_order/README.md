# Assuming Signal and Update Delivery Order

> [!TIP]
> [Signals](terms/signals.md) and [updates](terms/updates.md) may arrive in a different order than sent, even from a single client. Design workflows to tolerate any delivery order.

Developers often assume that sending signal A before signal B guarantees the workflow processes A first. This is incorrect. Network conditions, server-side concurrency, and client retries can all cause reordering. When workflow logic depends on specific ordering ("initialize" before "process"), out-of-order delivery leads to incorrect state or failures.

Design for any order: include sequence numbers in signal [payloads](terms/payload.md) and buffer out-of-order messages, use timestamps with last-write-wins resolution, model your workflow as a state machine that validates transitions, or send a single signal with a batch of ordered operations when ordering truly matters.

Unlike signals, updates can be *rejected* by the workflow via a validator function -- this lets you enforce ordering at the API boundary. For example, using [ULIDs](https://github.com/ulid/spec) (which are lexicographically sortable by time), the workflow can reject any update that doesn't have a higher ID than the last one it accepted:

<!--SNIPSTART assuming-signal-update-order-workflow-->
[assuming_signal_update_order/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_signal_update_order/workflow.go)
```go
// Change represents an update payload with a ULID for ordering.
type Change struct {
	ID   string // ULID
	Data string
}

func MyWorkflow(ctx workflow.Context) error {
	var lastID string
	var applied []Change

	err := workflow.SetUpdateHandlerWithOptions(ctx, "apply-change",
		func(ctx workflow.Context, c Change) error {
			lastID = c.ID
			applied = append(applied, c)
			return nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, c Change) error {
				if c.ID <= lastID {
					return fmt.Errorf("out-of-order update: %s <= %s", c.ID, lastID)
				}
				return nil
			},
		},
	)
	if err != nil {
		return err
	}

	// Wait for a "done" signal to complete the workflow.
	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)
	return nil
}

```
<!--SNIPEND-->

For workflows that aggregate multiple signals or updates, use property-based testing to verify robustness to any delivery order. The [`rapid`](https://pkg.go.dev/pgregory.net/rapid) library can generate random permutations of updates and assert that the workflow produces the same result regardless of order:

<!--SNIPSTART assuming-signal-update-order-test-->
[assuming_signal_update_order/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_signal_update_order/workflow_test.go)
```go
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

```
<!--SNIPEND-->
