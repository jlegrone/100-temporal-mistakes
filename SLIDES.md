---
title: "100 Temporal Mistakes"
subtitle: "And How to Avoid Them"
author: "Jacob LeGrone"
theme:
  background_color: "#FFFFFF"
  title_color: "#1E1E2E"
  text_color: "#333333"
  accent_color: "#7C3AED"
  code_bg_color: "#1E1E2E"
  code_text_color: "#D4D4D4"
  diff_add_color: "#10f300"
  diff_remove_color: "#f00000"
  font_heading: "Arial"
  font_body: "Arial"
  code_font: "Courier New"
---

# Introduction

<!-- QR code linking to slides in markdown format for those who want to follow along with code examples -->

---

# Part One: Activities

Activities are how Temporal workers interact with the outside world.

Activities have to deal with:
- Downstream service errors
- Worker crashes & hangs
- Temporal server disruptions

## Temporal's shared responsibility model for activities:

Temporal server provides an "at most once" execution semantic and retries activities by default.

It's on us to:
- Implement idempotency
- Not amplify bad requests
- Gracefully handle cancelation

---

## Activities: Handling Downstream Service Errors

<!-- Code for simple example activity that calls a generic payments API and returns the result (modeled after Stripe) -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
    httpReq := newPaymentReq(req) // POST api.example.com/v1/payments/charge

    resp, err := w.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

    // Decode the HTTP response and return
    // ...
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

    // 
    switch resp.StatusCode {
    case http.StatusBadRequest:
        return nil, temporal.NewNonRetryableApplicationError(resp.Status, "http_400", nil)
    }

    // Decode the HTTP response and return
    // ...
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

    // Decode the HTTP response and return
    // ...
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

---

## Activities: Weathering System Outages (continued)

<!-- Update the schedule to close timeout to 1h in the code example. Include code comment saying "allow retrying for up to 1 hour".

Speaker note: So the first timeout mistake to avoid is a schedule to close timeout that's too short. Pick a value based on how long you want to retry in the face of a serious system outage.

Speaker note: ScheduleToClose doesn't always need to be large. Long values (hours) make sense when the workflow should weather an extended outage and the caller is OK waiting, or is notified asynchronously. Short values (seconds to minutes) make sense when the workflow has a graceful degradation path, when reporting an error quickly is preferable to retrying through an outage, or when an upstream caller is waiting synchronously.
-->
<!-- TODO: make this a diff against the previous slide -->
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

## Activities: Weathering System Outages (continued)

<!-- Update code example, now adding a retry policy with MaxAttempts set to 3, initial backoff to 1s, backoff coefficient to 2, and max backoff to 30s. -->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour, // allow retrying for up to 1 hour
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

<!-- Speaker note: Another way an activity's retry behavior can be unexpectedly limited is by setting MaxAttempts. For example now if this activity quickly returns an error, we would exhaust all of our retries in less than 10 seconds, even though our intent was to survive outages of up to 1 hour.

Speaker note: For reference, the default activity retry policy uses exponential backoff with a 2.0 backoff coefficient, a 1-second initial interval, a 100-second maximum interval, and unlimited attempts. Source: https://docs.temporal.io/encyclopedia/retry-policies. Workflows have no default retry policy.
-->

---

## Activities: Weathering System Outages (continued)

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

---

## Activities: Idempotency Keys

<!-- Back to the previous payment code example. Update the code to compute an idempotency key (using activityhelpers.GetIdempotencyToken) and add it to the request header (follow the example from stripe docs: https://docs.stripe.com/api/idempotent_requests). -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargePaymentRequest) (*ChargePaymentResponse, error) {
    httpReq := newPaymentReq(req) // POST api.example.com/v1/payments/charge
    httpReq.Header.Set("Idempotency-Key", getIdempotencyToken(ctx))

    // ... send the HTTP request & handle errors
}

func getIdempotencyToken(ctx context.Context) string {
	info := activity.GetInfo(ctx)
	key := fmt.Sprintf("%s:%s:%s", info.WorkflowExecution.ID, info.WorkflowExecution.RunID, info.ActivityID)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
```

<!-- Speaker note: if you are lucky enough to be using an API that directly supports idempotency keys, whether in the form of a header or a client side request identifier, then you can also compute one based on the workflow and activity IDs. -->

---

## Activities: Natural Idempotency

<!-- TODO(jlegrone): rehearse the transition from the payments example to this k8s example so the use-case shift lands smoothly during the talk. -->

<!-- Speaker note: This activity is currently not idempotent because if the second API call fails, then when it's retried it will fail attempting to create a job with the same name instead of attaching to the existing one. -->

<!-- New code example: An activity called RunKubernetesJob that starts a k8s job and waits for it to complete (two k8s API calls). The activity should accept a struct with Name and Namespace fields, and return a struct with a Status field (completed or failed) -->
```go
func (w *Worker) RunKubernetesJob(ctx context.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    jobs := w.client.BatchV1().Jobs(req.Namespace)
    // Create the job
    if _, err := jobs.Create(ctx, &batchv1.Job{
        Name: req.Name,
    }); err != nil {
        return nil, err
    }

    // Poll for final status
    for {
        j, err := jobs.Get(ctx, req.Name)
        if err != nil { return nil, err }
        if status := getJobStatus(j); status.IsTerminal() {
            return &RunKubernetesJobResponse{Status: status}, nil
        }
        activity.RecordHeartbeat(ctx)
        time.Sleep(15 * time.Second)
    }
}
```

---

## Activities: Natural Idempotency (continued)

<!-- Updated code example, now ignoring an already exists error for the job. -->
```diff
 func (w *Worker) RunKubernetesJob(ctx context.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
     jobs := w.client.BatchV1().Jobs(req.Namespace)
-    // Create the job
+    // Create the job if it doesn't exist
     if _, err := jobs.Create(ctx, &batchv1.Job{
         Name: req.Name,
-    }); err != nil {
+    }); err != nil && !apierrors.IsAlreadyExists(err) {
         return nil, err
     }
 
     // Poll for final status
     // ...
 }
```

---

## Activities: Natural Idempotency (continued)
<!-- Update code example: Split into two activities, one called StartKubernetesJob and another called AwaitKubernetesJob. -->
```go
func (w *Worker) StartKubernetesJob(ctx context.Context, req StartKubernetesJobRequest) error {
    jobs := w.client.BatchV1().Jobs(req.Namespace)
    // Create the job if it doesn't exist
    _, err := jobs.Create(ctx, &batchv1.Job{ /* ... */ })
    if err != nil && !apierrors.IsAlreadyExists(err) {
        return err
    }
    return nil
}

func (w *Worker) AwaitKubernetesJob(ctx context.Context, req AwaitKubernetesJobRequest) (*AwaitKubernetesJobResponse, error) {
    // Poll for final status
    // ...
}
```

<!-- Speaker note: Now the workflow needs to call both activities, one after the other, but it doesn't matter how many times either of them is retried and we get more visibility into what's going on through the workflow history.

Speaker note (likely audience question -- "should activities be small? what about lots of activities?"):
- Fine-grained activities are usually a win: more visibility in workflow history, smaller retry scope, easier to make idempotent.
- A worker can register many activity types with no per-type runtime cost; the only direct cost of more activities is in workflow history (each invocation adds events).
- The tradeoff comes back in the next section -- workflow history size limits.
-->

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

---

## Activities: Handling Worker Disruptions (continued)

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

---

## Activities: Handling Worker Disruptions (continued)

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

- Activities SHOULD be idempotent. Temporal can re-execute an activity at least once during failover or retry, so any side effect needs to be safe to repeat (or naturally idempotent, like a read).
- ALWAYS set a **ScheduleToClose** timeout. Base the value on how long the activity should continue retrying during a worst case outage.
- ALWAYS set EITHER **Heartbeat** OR **StartToClose** timeout. Use **StartToClose** timeout only when the activity is guaranteed to not run past that duration and it is acceptable to wait the whole duration before a retry. 
- Activities that perform cleanup on cancelation MUST send heartbeats.
- PREFER unlimited attempts with `ScheduleToClose` as the bound. Reserve `MaxAttempts` for cases where each attempt has external cost -- account lockouts, alert fatigue, dispute thresholds.
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

## Workflows: Living Within Server Limits

<!-- No code example -- just the limits. Numbers sourced from src/overflowing-*.md. -->

Server-imposed limits to be aware of:
- **Individual payload size**: ~4MB per workflow/activity input or output, signal, or update (inherited from the Temporal server's gRPC message limit).
- **Workflow history bytes**: 50MB (sum of all events in the workflow). Results in termination.
- **Workflow history length**: 50,000 events. Results in termination.
- **Workflow task timeout**: 10 seconds (per workflow task -- not the workflow execution timeout). Results in failed workflow task (retried).

Mitigations:
- Use **ContinueAsNew** to start a fresh execution with reset history. Check `GetContinueAsNewSuggested()` to know when the server is recommending it.
- Avoid passing large payloads from activities to workflows, or use external payload storage.

---

## Workflows: Keeping Code Deterministic

<!-- No code example here -- just orient the audience to the categories of non-determinism they'll need to watch out for. Sourced from src/terms/non-determinism.md. -->

Common sources of non-determinism in workflow code:
- Network calls (HTTP, DB queries, gRPC) -- re-execute on every replay and may return different results
- System time (`time.Now()` vs `workflow.Now()`)
- Random number generation
- Usage of environment variables
- Interactions with the filesystem
- Goroutines spawned outside `workflow.Go` -- the SDK doesn't record their scheduling, and they often race with the workflow function for shared state
- Variable references from outside the workflow function scope

---

## Workflows: Keeping Code Deterministic (continued)

Tools to catch non-determinism:
- **Go**: [`workflowcheck`](https://github.com/temporalio/sdk-go/tree/master/contrib/tools/workflowcheck) -- opt-in static analyzer that flags non-deterministic calls in workflow code.
- **Python**: [Workflow sandbox](https://docs.temporal.io/develop/python/python-sdk-sandbox) -- enabled by default; restricts imports and module access at runtime.
- **TypeScript**: V8 isolate sandboxing is built-in -- workflow code runs in a separate V8 context with no Node.js APIs.

---

## Workflows: Versioning Code Changes

<!-- Speaker notes:
- Versioning is required for *any* change to workflow code that would result in a different workflow history when it runs against an existing execution.
- Common changes that trigger this:
  - Adding, removing, or reordering activities, child workflows, signals, or timers -- anything that changes the recorded command sequence
  - Changing activity arguments or activity options
  - Adding or changing a `workflow.SideEffect`
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

     return workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
 }
```

<!-- Speaker notes: GetVersion is reached only inside a conditional branch some workflows never enter. The TemporalChangeVersion search attribute is never set on those executions, so a list-workflow query filtering by version keeps returning unversioned workflows indefinitely. 

Workflows that never take this branch will NEVER set the TemporalChangeVersion search attribute; you can't tell from a list query whether they're safe to clean up.
-->

---

## Workflows: Evaluate Change Versions Up Front

<!-- Speaker note: The fix is to hoist the version check to the top of the workflow so every execution records the version as soon as it starts, even if the branch it gates is never taken. -->
```diff
 func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
+    inventoryResVersion := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 1)

     // ... generate a charge request for the item & customer

+    switch inventoryResVersion {
+    case 1:
+        if req.RequiresInventoryReservation {
+            if err := workflowhelpers.AwaitActivity(ctx, w.ReserveInventory, reserveRequest); err != nil {
+                return nil, err
+            }
+        }
+    }

     return workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
 }
```

---

## Workflows: Evaluate Change Versions Up Front (continued)

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

## Workflows: Verifying Replay Safety (continued)

<!-- Speaker note: Run replay tests in CI against the captured fixtures. If the new code's command sequence diverges from any recorded history, the test fails before the change reaches production. Pair this with `workflowcheck` static analysis to catch the obvious sources of non-determinism. -->

```go
func TestReplayWorkflowHistory(t *testing.T) {
    testhelpers.AssertWorkflowReplayFromJSONFiles(t, 
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

## Workflows: Cleaning Up Patches (continued)

<!-- Speaker note: With the count at 0, it's safe to delete the v0 and v1 branches. Bumping the min supported version to 2 keeps the GetVersion call (so existing v2 histories still replay) while making replay fail with an
explicit error if a stale v0/v1 history ever shows up. -->

```diff
 func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
-    inventoryResVersion := workflow.GetVersion(ctx, "add-reserve-inventory", workflow.DefaultVersion, 2)
+    // Bump min supported version to 2 so v2 histories continue to replay,
+    // and any leftover v0/v1 history fails loudly instead of silently diverging.
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

<!-- Speaker notes: A Selector fans in shipping, a fulfilment deadline, and ctx cancelation; whichever fires sets err. On err, the workflow tries to refund via a child workflow.

But this naive version uses the parent's (possibly canceled) ctx, the default ParentClosePolicy, and doesn't wait for the child to be scheduled before returning. Each of those is a bug we'll fix on the next slide. -->

<!-- TODO: Rework this example to wait for a signal that the shipment has completed instead of running an activity. That sets up the cancelation deadlock bug. -->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... reserve inventory and charge payment

    shipFuture := workflow.ExecuteActivity(ctx, w.ShipItem, shipRequest)
    sel := workflow.NewNamedSelector(ctx, "shipment")

    sel.AddFuture(shipFuture, func(f workflow.Future) { err = f.Get(ctx, &shipmentResponse) })
    sel.AddFuture(workflow.NewTimer(ctx, 12*time.Hour), func(f workflow.Future) {
        err = workflow.ErrDeadlineExceeded
    })
    sel.AddReceive(ctx.Done(), func(c workflow.ReceiveChannel, more bool) { err = ctx.Err() })

    sel.Select(ctx)
    if err != nil {
        // BAD: shares the parent's ctx, no abandon policy, doesn't await scheduling.
        workflow.ExecuteChildWorkflow(ctx, w.RefundPayment, refundRequest)
        return nil, err
    }

    return &PurchaseItemResponse{TrackingID: shipmentResponse.TrackingID}, nil
}
```

---

## Workflows: Designing for Cancelation

<!-- Speaker notes: Three changes make the refund actually compensate the customer.

1. workflow.NewDisconnectedContext detaches the cleanup from the parent's cancelation, so the refund command can still be issued.
2. ParentClosePolicy ABANDON keeps the child running after the parent closes, so the refund completes even if the parent returns immediately.
3. GetChildWorkflowExecution().Get blocks until the server has accepted the start command, so we know the child is durably scheduled before the parent returns.

Speaker note: This API is Go-specific (workflow.NewDisconnectedContext). Other SDKs have equivalent mechanisms under different names -- the concept of decoupling cleanup from parent cancelation is universal.
-->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ...

    sel.Select(ctx)
    if err != nil {
        if startErr := w.startDisconnectedRefundWorkflow(ctx, refundRequest); startErr != nil {
            workflow.GetLogger(ctx).Warn("failed to start refund", "error", startErr)
        }
        return nil, err
    }

    // ...
}

func (w *Worker) startDisconnectedRefundWorkflow(ctx workflow.Context, req RefundRequest) error {
    ctx = workflow.WithChildOptions(workflow.ChildWorkflowOptions{
        ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON // "ABANDON" parent close policy
    })
    ctx, _ = workflow.NewDisconnectedContext(ctx) // Disconnected workflow context
    fut := workflow.ExecuteChildWorkflow(ctx, w.RefundPayment, req)
    return fut.GetChildWorkflowExecution().Get(ctx, nil) // Block until child workflow start
}
```

---

## Workflows: Use Internal Timers for Compensation

> [!TIP]
> When a workflow execution timeout fires, Temporal *terminates* the workflow -- it does not *cancel* it. No deferred functions run, no cancelation handlers fire. If you need a chance to compensate, build the deadline yourself with an internal timer.

```go
// Reserve at least 1 minute for compensation before the hard timeout.
softTimeout, err := getSoftTimeout(ctx, time.Minute)
if err != nil {
    return err
}
// ... race the soft timeout against child completion and ctx.Done()
```

Source: `src/assuming_workflow_timeouts_allow_graceful_cleanup/`

---

## Workflows: Cap Workflow Lifetime With ContinueAsNew

> [!TIP]
> Long-running workflows accumulate history events, lengthening replay time and eventually hitting the 50k event limit. Cap lifetime by completing or calling **ContinueAsNew** within 24 hours.

```go
if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
    return workflow.NewContinueAsNewError(ctx, MyWorkflow, state)
}
```

Trigger ContinueAsNew on event count, elapsed time (24 h caps code age and simplifies versioning), or an explicit signal for operational control.

Source: `src/not_using_continue_as_new/`

---

## Workflows: Drain Signals Before Completing

> [!TIP]
> If a workflow completes or calls ContinueAsNew while signals are buffered in its channel, those signals are silently lost. Drain the channel before returning.

```go
for {
    var signal MySignal
    if ok := signalCh.ReceiveAsync(&signal); !ok {
        break
    }
    state.Apply(signal)
}
return workflow.NewContinueAsNewError(ctx, MyWorkflow, state)
```

Apply the drained signals to your state -- don't just read and discard them. Make sure all completion paths (success, error, ContinueAsNew) include draining.

Source: `src/not_draining_signals_before_completing_workflow/`

---

## Workflows: Fan Out Large Batches to Child Workflows

> [!TIP]
> Concentrating all batch work in one workflow causes history overflow and lock contention. Distribute work across child workflows instead.

Activities scheduled concurrently compete for the workflow lock, and large histories push against the 50k event limit. Split work into size-limited batches and handle each batch in a separate child workflow. Nest as needed -- batches of batches -- to create a tree with concurrency control at each level.

Source: `src/naive-batch-processing-implementation.md`

---

## Workers: Worker Versioning and Pinned Workflows

> [!TIP]
> Worker versioning with pinned workflows ensures each running workflow replays against the same code version it started on -- removing most of the patching burden as deployments roll forward.

<!-- TODO(jlegrone): backfill from a dedicated mistake entry once one is written; for now, link the official docs. -->

Reference: https://docs.temporal.io/worker-versioning

---

## Workers: External Payload Storage for Large Payloads

> [!TIP]
> Individual workflow/activity request/response payloads, signals, and updates cannot exceed 4 MB by default (inherited from the Temporal server's gRPC limit). Use external storage when larger data must flow through workflow code.

Trim inputs and outputs to the minimum the caller needs. Store large data in an external system (database, blob storage, sessions) and pass references (IDs, URLs) instead. When larger payloads are genuinely required, an external storage codec can offload them at the serialization layer.

Reference: https://docs.temporal.io/external-storage  
Source: `src/overflowing-maximum-individual-payload-size.md`

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
