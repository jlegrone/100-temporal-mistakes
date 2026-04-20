# Assuming Signal and Update Delivery Order

> [!TIP]
> [Signals](terms/signals.md) and [updates](terms/updates.md) may arrive in a different order than sent, even from a single client. Design workflows to tolerate any delivery order.

Developers often assume that sending signal A before signal B guarantees the workflow processes A first. This is incorrect. Network conditions, server-side concurrency, and client retries can all cause reordering. When workflow logic depends on specific ordering ("initialize" before "process"), out-of-order delivery leads to incorrect state or failures.

Design for any order: include sequence numbers in signal [payloads](terms/payload.md) and buffer out-of-order messages, use timestamps with last-write-wins resolution, model your workflow as a state machine that validates transitions, or send a single signal with a batch of ordered operations when ordering truly matters.

A naive workflow that accepts updates without validation will silently apply them in whatever order they arrive:

<!--SNIPSTART assuming-signal-update-order-workflow-v1-->
[assuming_signal_update_order/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_signal_update_order/workflow.go)
```go

// MyWorkflowV1 accepts all updates regardless of order.
// If updates arrive out of order, the workflow silently applies them
// in whatever order they were delivered.
func MyWorkflowV1(ctx workflow.Context) (string, error) {
	var lastData string

	err := workflow.SetUpdateHandler(ctx, "apply-change",
		func(ctx workflow.Context, c Change) error {
			lastData = c.Data
			return nil
		},
	)
	if err != nil {
		return "", err
	}

	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)

	return lastData, nil
}

```
<!--SNIPEND-->

Unlike signals, updates can be *rejected* by the workflow via a validator function -- this lets you enforce ordering at the API boundary. For example, using [ULIDs](https://github.com/ulid/spec) (which are lexicographically sortable by time), the workflow can reject any update that doesn't have a higher ID than the last one it accepted:

<!--SNIPSTART assuming-signal-update-order-workflow-v2-->
[assuming_signal_update_order/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_signal_update_order/workflow.go)
```go

// MyWorkflowV2 rejects out-of-order updates using a ULID-based validator.
func MyWorkflowV2(ctx workflow.Context) (string, error) {
	var lastID string
	var lastData string

	err := workflow.SetUpdateHandlerWithOptions(ctx, "apply-change",
		func(ctx workflow.Context, c Change) error {
			lastID = c.ID
			lastData = c.Data
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
		return "", err
	}

	workflow.GetSignalChannel(ctx, "done").Receive(ctx, nil)

	return lastData, nil
}

```
<!--SNIPEND-->

For workflows that aggregate multiple signals or updates, use property-based testing to verify robustness to any delivery order. For example in Go, the [`rapid`](https://pkg.go.dev/pgregory.net/rapid) library can generate random permutations of updates and assert that the workflow produces the same result regardless of order:

<!--SNIPSTART assuming-signal-update-order-test-->
[assuming_signal_update_order/workflow_test.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/assuming_signal_update_order/workflow_test.go)
```go
func TestWorkflowOrderInvariance(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		env := testsuite.NewTestWorkflowEnvironment(t)

		// Generate changes with increasing ULIDs.
		var changes []Change
		for range rapid.IntRange(0, 19).Draw(t, "n") {
			changes = append(changes, Change{
				ID:   ulid.Make().String(),
				Data: rapid.String().Draw(t, "data"),
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

```
<!--SNIPEND-->
