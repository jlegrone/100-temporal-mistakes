# Introduction

<!-- QR code linking to slides in markdown format for those who want to follow along with code examples -->

---

# Part One: Activities

Need to be robust to:
- Downstream service errors
- Worker crashes & hangs
- Temporal server disruption

Should also:
- Not amplify bad requests
- Gracefully handle cancelation

---

## Activities: Handling Downstream Service Errors

<!-- Code for simple example activity that calls an generic payments API and returns the result (modeled after Stripe) -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
    httpReq := newPaymentReq(req) // POST api.example.com/v1/payments/charge

    resp, err := w.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

    // ... decode the HTTP response and return
}
```

---

## Activities: Avoid Amplifying Invalid Requests

<!-- Updated code example that inspects http status code and returns non-retryable TemporalApplicationError for bad requests (HTTP 400) (use switch statement for HTTP status so more cases can easily be added in the future) -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
    httpReq := newPaymentReq(req) // POST api.example.com/v1/payments/charge

    resp, err := w.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

    switch resp.StatusCode {
    case http.StatusBadRequest:
        return nil, temporal.NewNonRetryableApplicationError(resp.Status, "http_400", nil)
    }

    // ... decode the HTTP response and return
}
```

---

## Activities: Avoid Overloading Services With Retries

<!-- Updated code example that also increases the next retry backoff time when external service returns a resource overloaded error (HTTP 429) using the activityhelpers.GetNextRetryDelay function with a minimum backoff coefficient of 3, so retries against the rate-limited endpoint back off more aggressively than the workflow's default policy. -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
    // ... build and send the HTTP request

    switch resp.StatusCode {
    case http.StatusBadRequest:
        return nil, temporal.NewNonRetryableApplicationError(resp.Status, "http_400", nil)
    case http.StatusTooManyRequests:
        // Increase the retry delay
        delay := activityhelpers.GetNextRetryDelay(ctx, 3)
        return nil, temporal.NewApplicationErrorWithOptions(resp.Status, "http_429",
            temporal.ApplicationErrorOptions{NextRetryDelay: delay})
    }

    // ... decode the HTTP response and return
}
```

---

## Activities: Weathering System Outages

<!-- New code example, this time showing the workflow code that invokes the payment activity. Set a 30s start to close timeout and a 1m schedule to close timeout. -->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Minute,
    })
    resp, err := workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)

    // ...
}
```

<!-- Speaker note: Temporal is great at retrying activities, but it's still important to think carefully about how we configure timeouts and retry policies in order to survive worst case system outages. For example here I'm invoking my activity with a schedule to close timeout that doesn't give much room for the activity to be retried if the worker or downstream API are temporarily unavailble. -->

<!-- Update the schedule to close timeout to 1h in the code example. Include code comment saying "allow retrying for up to 1 hour".

Speaker note: So the first timeout mistake to avoid is a schedule to close timeout that's too short. Pick a value based on how long you want to retry in the face of a serious system outage.
-->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour, // allow retrying for up to 1 hour
    })

    resp, err := workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
    // ...
}
```

---

## Activities: Weathering System Outages

<!-- Update code example, now adding a retry policy with MaxAttempts set to 3, initial backoff to 1s, backoff coefficient to 2, and max backoff to 30s. -->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts:    3,
            InitialInterval:    time.Second,
            BackoffCoefficient: 2,
            MaximumInterval:    30 * time.Second,
        },
    })
    
    // ... execute the ChargePayment activity
}
```

<!-- Speaker note: Another way an activity's retry behavior can be unexpectedly limited is by setting MaxAttempts. For example now if this activity quickly returns an error, we would exhaust all of our retries in less than 10 seconds, even though our intent was to survive outages of up to 1 hour. -->

<!-- Comment out the MaxAttempts field in the code example. Include code comment saying "allow unlimited attempts until the ScheduleToClose timeout is reached".

Speaker note: I think a much simpler mental model is to allow unlimited attempts, and only use retry policy to tune the backoff behavior like initial interval and maximum backoff duration as needed.
-->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
        RetryPolicy: &temporal.RetryPolicy{
            // MaximumAttempts: 0, // allow unlimited attempts until the ScheduleToClose timeout is reached
            InitialInterval:    time.Second,
            BackoffCoefficient: 2,
            MaximumInterval:    30 * time.Second,
        },
    })

    // ... execute the ChargePayment activity
}
```

---

## Activities: Implementing Idempotency

Three techniques to achieve idempotency:
- Passing idempotency key to external APIs
    - Derive a key from the Workflow ID and activity ID. Pass this to downstream systems (like Stripe) to ignore duplicate requests.
- Applying database constraints
    - Use `INSERT ... ON CONFLICT` or conditional writes to ensure records aren't created twice.
- Using naturally idempotent operations
    - Design side effects as state settings (Set to X) rather than increments (+1), or use upserts with fixed IDs.
    - May help to decompose into multiple activities.

## Activities: Idempotency Keys

<!-- Back to the previous payment code example. Update the code to compute an idempotency key (using activityhelpers.GetIdempotencyToken) and add it to the request header (follow the example from stripe docs: https://docs.stripe.com/api/idempotent_requests). -->
```go
func getIdempotencyToken(ctx context.Context) string {
	info := activity.GetInfo(ctx)
	key := fmt.Sprintf("%s:%s:%s", info.WorkflowExecution.ID, info.WorkflowExecution.RunID, info.ActivityID)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}

func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
    httpReq := newPaymentReq(req) // POST api.example.com/v1/payments/charge
    httpReq.Header.Set("Idempotency-Key", getIdempotencyToken(ctx))

    // ... send the HTTP request & handle errors
}
```

<!-- Speaker note: if you are lucky enough to be using an API that directly supports idempotency keys, whether in the form of a header or a client side request identifier, then you can also compute one based on the workflow and activity IDs. -->

---

## Activities: Natural Idempotency

<!-- New code example: An activity called RunKubernetesJob that starts a k8s job and waits for it to complete (two k8s API calls). The activity should accept a struct with Name and Namespace fields, and return a struct with a Status field (completed or failed) -->
```go
func (w *Worker) RunKubernetesJob(ctx context.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    jobs := w.client.BatchV1().Jobs(req.Namespace)
    if _, err := jobs.Create(ctx, &batchv1.Job{
        Name: req.Name,
    }); err != nil {
        return nil, err
    }

    cancel := activityhelpers.AutoHeartbeat(ctx)
    defer cancel()

    // Poll for final status
    for {
        j, err := jobs.Get(ctx, req.Name)
        if err != nil { return nil, err }
        if status := getJobStatus(j); status.IsTerminal() {
            return &RunKubernetesJobResponse{Status: status}, nil
        }
        time.Sleep(15*time.Second)
    }
}
```

<!-- Speaker note: This activity is currently not idempotent because if the second API call fails, then when it's retried it will fail attempting to create a job with the same name instead of attaching to the existing one. -->

<!-- Update code example: Split into two activities, one called StartKubernetesJob and another called AwaitKubernetesJob. -->
```go
func (w *Worker) StartKubernetesJob(ctx context.Context, req StartKubernetesJobRequest) error {
    _, err := w.client.BatchV1().Jobs(req.Namespace).Create(ctx, &batchv1.Job{ /* ... */ })
    // Ignore already exists errors; the activity has already run at least once.
    if err != nil && !apierrors.IsAlreadyExists(err) {
        return err
    }
    return nil
}

func (w *Worker) AwaitKubernetesJob(ctx context.Context, req AwaitKubernetesJobRequest) (*AwaitKubernetesJobResponse, error) {
    // ... Poll for final status
}
```

<!-- Speaker note: Now the workflow needs to call both activities, one after the other, but it doesn't matter how many times either of them is retried and we get more visibility into what's going on through the workflow history. -->

---

## Activities: Handling Worker Disruptions

Choosing between StartToClose and Heartbeat timeouts

<!-- New workflow code example, this time invoking our (longer running) AwaitKubernetesJob activity. Set a 30s start to close timeout and a 1h schedule to close timeout. -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    // Start the job
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            StartToCloseTimeout:    30 * time.Second,
            ScheduleToCloseTimeout: time.Hour,
        }),
        w.AwaitKubernetesJob,
        AwaitKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

<!-- Speaker note: Our previous activity example was expected to always complete in under 30s. But that's not the case for all activities. A short start to close timeout made sense for the payments use case, but what about waiting for a k8s job that could run for much longer? Setting too short a value could mean that some activities never complete, no matter how many retry attempts are made.

So we can try increasing the start to close timeout, but now this also means that if the worker crashes or becomes unresponsive, we'd have to wait much longer before Temporal retries the activity.
-->

<!-- Updated code example: Change the start to close timeout to 5m. -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    // Start the job
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            // Allow the activity to poll the job status for up to five minutes before timing out.
            StartToCloseTimeout:    5 * time.Minute,
            ScheduleToCloseTimeout: time.Hour,
        }),
        w.AwaitKubernetesJob,
        AwaitKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

<!-- Updated code example: Replace the start to close timeout with a 30s heartbeat timeout.

Speaker notes:
- Replacing a start to close timeout with heartbeat timeout avoids the tradeoff between retrying quickly when the worker fails, and allowing your longest-running tasks to complete. Now the activity can run as long as it needs to, up to the schedule to close timeout, but is retried quickly if the worker becomes unresponsive.
 -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    // Start the job
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            // This activity may run for a long time, so use Heartbeat instead of StartToClose timeout.
            HeartbeatTimeout:       30 * time.Second,
            ScheduleToCloseTimeout: time.Hour,
        }),
        w.AwaitKubernetesJob,
        AwaitKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

---

## Activities: A Grand Unified Theory

- Activities MUST be idempotent.
- ALWAYS set a **ScheduleToClose** timeout. Base the value on how long the activity should continue retrying during a worst case outage.
- ALWAYS set EITHER **Heartbeat** OR **StartToClose** timeout. Use **StartToClose** timeout only when the activity is guaranteed to not run past that duration and it is acceptable to wait the whole duration before a retry. 
- Activities that perform cleanup on cancelation MUST send heartbeats.
- NEVER set `MaxAttempts` in your activity retry policy. Instead use `ScheduleToClose` timeout and backoff configuration to tune retry behavior and duration.
- Respect standard error codes from downstream services. Translate these into `TemporalApplicationError` to skip retry or adjust backoff behavior as needed.

** Consider implementing and/or enforcing these policies in an interceptor.
<!-- TODO: Add QR code linking to interceptor example from 100-temporal-mistakes repo. -->

---

# Part Two: Workflows

Need to be robust to:
- Workflow code changing across deployments
- Server-imposed history & payload limits
- Signals arriving in unpredictable order
- Cancelation requests at any point in execution

Should also:
- Stay deterministic across replays
- Yield quickly to the workflow task event loop

---

## Workflows: Keeping Code Deterministic

<!-- No code example here -- just orient the audience to the categories of non-determinism they'll need to watch out for. Sourced from src/terms/non-determinism.md. -->

Common sources of non-determinism in workflow code:
- Network calls
- System time (`time.Now()` vs `workflow.Now()`)
- Random number generation
- Usage of environment variables
- Interactions with the filesystem
- Coroutines not managed by the Temporal SDK
- Variable references from outside the workflow function scope

---

## Workflows: Keeping Code Deterministic

<!-- TODO: Link to documentation on workflowcheck and sandboxes in typescript and python SDKs -->

---

## Workflows: Versioning Code Changes

<!-- Speaker notes:
- Versioning is required for *any* change to workflow code that would result in a different workflow history when it runs against an existing execution.
- Common changes that trigger this:
  - Adding, removing, or reordering activities or child workflows
  - Adding a timer
  - Rejecting an update
-->
```diff
 func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
     // ... generate a charge request for the item & customer

+    if req.RequiresInventoryReservation {
+        v := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 1)
+        if v == 1 {
+            if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
+                return nil, err
+            }
+        }
+    }
+
     return workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
 }
```

<!-- Speaker notes: GetVersion is reached only inside a conditional branch some workflows never enter. The TemporalChangeVersion search attribute is never set on those executions, so a list-workflow query filtering by version keeps returning unversioned workflows indefinitely. 

Workflows that never take this branch will NEVER set the TemporalChangeVersion search attribute; you can't tell from a list query whether they're safe to clean up.
-->

---

## Workflows: Evaluate Patches Up Front

<!-- Speaker note: The fix is to hoist the version check to the top of the workflow so every execution records the version as soon as it starts, even if the branch it gates is never taken. -->
```diff
 func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
+    inventoryResVersion := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 1)
+
     // ... generate a charge request for the item & customer

+    switch inventoryResVersion {
+    case 1:
+        if req.RequiresInventoryReservation {
+            if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
+                return nil, err
+            }
+        }
+    }
+
     return workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
 }
```

---

## Workflows: Evaluate Patches Up Front

<!-- Speaker note: Subsequent changes bump the patch's max version. The decision of whether to reserve inventory now moves into the activity, so the workflow always calls it on the new code path. In-flight workflows that started under v1 keep following `case 1`; new workflows take `case 2`. Once all `case 1` executions have closed, you can remove that branch but keep the GetVersion call (and bump the min compatible version) so replayed v1 histories still resolve. -->
```diff
 func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
-    inventoryResVersion := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 1)
+    inventoryResVersion := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 2)

     // ... generate a charge request for the item & customer

     switch inventoryResVersion {
     case 1:
         if req.RequiresInventoryReservation {
             // ... await ReserveInventory activity
         }
+    case 2:
+        // The activity now decides internally whether to reserve.
+        if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
+            return nil, err
+        }
     }

     return workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
 }
```

---

## Workflows: Verifying Replay Safety

<!-- Speaker note: Capture a representative history for each patch branch. Use the TemporalChangeVersion search attribute to bucket existing executions, and pick the earliest one in each bucket so the fixture exercises the most history. -->

```bash
# Find the earliest workflow that did not hit either patch branch
temporal workflow list \
  --query 'WorkflowType="PurchaseItem" AND TemporalChangeVersion NOT IN ("add-reserve-inventory-1", "add-reserve-inventory-2")' \
  --order-by 'StartTime ASC' --limit 1

# Find the earliest workflow that took version 1 of the patch.
temporal workflow list \
  --query 'WorkflowType="PurchaseItem" AND TemporalChangeVersion IN ("add-reserve-inventory-1")' \
  --order-by 'StartTime ASC' --limit 1

# Download workflow history to a test fixture path.
temporal workflow show --workflow-id <ID> --output json \
  > testdata/purchase_item_history_<PATCH_VERSION>.json
```

---

## Workflows: Verifying Replay Safety

<!-- Speaker note: Run replay tests in CI against the captured fixtures. If the new code's command sequence diverges from any recorded history, the test fails before the change reaches production. Pair this with `workflowcheck` static analysis to catch the obvious sources of non-determinism. -->

```go
func TestReplayWorkflowHistory(t *testing.T) {
    testsuite.AssertWorkflowReplayFromJSONFiles(t, 
        PurchaseItem,
        "testdata/purchase_item_history_v0.json",
        "testdata/purchase_item_history_v1.json",
    )
}
```

** Check the code coverage for the version branches in your workflow! If they aren't covered, then the replay test is not validating your change.

More techniques: https://temporal.io/resources/on-demand/replay-safety-at-datadog

---

## Workflows: Cleaning Up Patches

<!-- Speaker note: Once every running workflow has either completed or evaluated v2, the v0 and v1 branches can be deleted. Wait until this count returns 0 -- the 5-minute cutoff avoids false positives for workflows that have been started but have not yet persisted the search attribute. -->

```bash
# Returns 0 when no in-flight workflow can still be on v0 or v1.
# Use `date` on Linux
CUTOFF=$(gdate -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S.%3NZ)
temporal workflow count \
  --query "WorkflowType='PurchaseItem' AND ExecutionStatus='Running' AND TemporalChangeVersion NOT IN ('add-reserve-inventory-2') AND StartTime < '$CUTOFF'"
```

---

## Workflows: Cleaning Up Patches

<!-- Speaker note: With the count at 0, it's safe to delete the v0 and v1 branches. Bumping the min supported version to 2 keeps the GetVersion call (so existing v2 histories still replay) while making replay fail with an
explicit error if a stale v0/v1 history ever shows up. -->

```diff
 func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
-    inventoryResVersion := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 2)
+    workflow.GetVersion(ctx, "add-reserve-inventory", 2, 2)

-    switch inventoryResVersion {
-    case 1:
-        if req.RequiresInventoryReservation {
-            if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
-                return nil, err
-            }
-        }
-    case 2:
-        // The activity decides internally whether to reserve.
-        if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
-            return nil, err
-        }
+    if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
+        return nil, err
     }
 }
```

---

## Workflows: Designing for Cancelation

<!-- Bad example: cleanup activity called with the same canceled ctx -- it's never scheduled because the context is already canceled. Also show a defer cleanup() blocked on a channel that never returns. -->
```go
func (w *Worker) OrderWorkflow(ctx workflow.Context, req OrderWorkflowRequest) (*OrderWorkflowResponse, error) {
    // ... place the order

    defer func() {
        // BAD: ctx may already be canceled here, so RefundPayment never runs.
        _ = workflowhelpers.AwaitActivity(ctx, w.RefundPayment, refundRequest)
    }()

    return w.runOrder(ctx, req)
}
```

<!-- Fix: workflow.NewDisconnectedContext for the cleanup activity. Show structural select on ctx.Done() plus signal channel for the deadlock case. -->
```go
func (w *Worker) OrderWorkflow(ctx workflow.Context, req OrderWorkflowRequest) (*OrderWorkflowResponse, error) {
    // ... place the order

    defer func() {
        cleanupCtx, cancel := workflow.NewDisconnectedContext(ctx)
        defer cancel()
        _ = workflowhelpers.AwaitActivity(cleanupCtx, w.RefundPayment, refundRequest)
    }()

    return w.runOrder(ctx, req)
}
```

---

## Workflows: Don't Block the Task Loop

<!-- Bad example: tight polling loop in workflow code racks up history events. -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    // ... start the job

    for {
        var resp PollJobStatusResponse
        if err := workflowhelpers.AwaitActivity(ctx, w.PollJobStatus, pollRequest, &resp); err != nil {
            return nil, err
        }
        if resp.Status.IsTerminal() {
            return &RunKubernetesJobResponse{Status: resp.Status}, nil
        }
        _ = workflow.Sleep(ctx, time.Second) // Each iteration adds events to history.
    }
}
```

<!-- Fix: replace the loop with a single long-running heartbeating activity that polls internally (this is the AwaitKubernetesJob activity from Part One). Briefly mention that clients polling for workflow results should use WorkflowRun.Get instead. -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    // ... start the job

    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            HeartbeatTimeout:       30 * time.Second,
            ScheduleToCloseTimeout: time.Hour,
        }),
        w.AwaitKubernetesJob,
        AwaitKubernetesJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

---

## Workflows: Living Within Server Limits

<!-- No code example -- just the limits. Numbers sourced from src/overflowing-*.md. -->

Server-imposed limits to be aware of:
- **Individual payload size**: ~4MB per workflow/activity input or output, signal, or update (inherited from the Temporal server's gRPC message limit).
- **Workflow history bytes**: 50MB (sum of all events in the workflow). Results in termination.
- **Workflow history length**: 50,000 events. Results in termination.
- **Workflow task timeout**: 10 seconds. Results in failed workflow task (retried).

Mitigations:
- Use `GetContinueAsNewSuggested()` to start a new workflow execution before hitting history size limits.
- Avoid passing large payloads from activities to workflows, or use external payload storage.

---

## Workflows: Composing Workflows

<!-- Bad example: a MonthlyBilling workflow that reads N customer rows and processes them all inline, bloating history and tying success to a single execution. -->
```go
func (w *Worker) MonthlyBilling(ctx workflow.Context, req MonthlyBillingRequest) error {
    var customers ListCustomersResponse
    if err := workflowhelpers.AwaitActivity(ctx, w.ListCustomers, listRequest, &customers); err != nil {
        return err
    }
    for _, c := range customers.IDs {
        if err := workflowhelpers.AwaitActivity(ctx, w.BillCustomer, BillCustomerRequest{ID: c}); err != nil {
            return err
        }
    }
    return nil
}
```

<!-- Fix: parent fans out to BillCustomer child workflows with deterministic IDs, ParentClosePolicy ABANDON for fire-and-forget, GetChildWorkflowExecution awaited before parent returns. Brief callout on workflow ID scoping (not-properly-scoping-semantic-workflow-ids). -->
```go
func (w *Worker) MonthlyBilling(ctx workflow.Context, req MonthlyBillingRequest) error {
    // ... list customers

    futures := make([]workflow.ChildWorkflowFuture, 0, len(customers.IDs))
    for _, c := range customers.IDs {
        childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
            WorkflowID:        fmt.Sprintf("billing-%s-%s", c, req.Month),
            ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
        })
        f := workflow.ExecuteChildWorkflow(childCtx, w.BillCustomer, BillCustomerRequest{ID: c})
        if err := f.GetChildWorkflowExecution().Get(ctx, nil); err != nil {
            return err
        }
        futures = append(futures, f)
    }
    // ... optionally wait on all futures
    return nil
}
```

---

## Workflows: Design Guidelines

Workflow functions:
- MUST be deterministic. Use the Temporal SDK for time, randomness, and side effects.
- MUST evaluate all patches as the first step (at the top of the function).
- MUST use internal timers rather than execution timeouts if they need to run compensating actions.[1]
- SHOULD be designed to complete or ContinueAsNew within 24 hours or when the server suggests ContinueAsNew.
- SHOULD drain all signals before completing or ContinueAsNew.
- SHOULD fan out large batches of work to child workflows.

Temporal workers:
- SHOULD have a replay testing harness.
- SHOULD be onboarded to worker versioning and pinned workflows.
- SHOULD enable external payload storage if activity responses are ~1MB or larger.

1. OR use child workflows with ParentClosePolicy of RequestCancel and Signal sentinel pattern.
