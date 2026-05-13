Content from presentation at Replay Conference in San Francisco May 7th 2026.

# 100 Temporal Mistakes

## What we'll cover
- Activities: Errors, retries, idempotency, timeouts, worker disruption
- Workflows: Runtime limitations, determinism, change versioning
- Recommendations to simplify working with Temporal day to day

## Table of Contents

- [Part One: Activities](#part-one-activities)
  - [Temporal's shared responsibility model for activities](#temporals-shared-responsibility-model-for-activities)
  - [Handling Downstream Service Errors](#activities-handling-downstream-service-errors)
  - [Avoid Amplifying Invalid Requests](#activities-avoid-amplifying-invalid-requests)
  - [Avoid Overloading Services With Retries](#activities-avoid-overloading-services-with-retries)
  - [Weathering System Outages](#activities-weathering-system-outages)
  - [Implementing Idempotency](#activities-implementing-idempotency)
    - [Idempotency Keys](#activities-idempotency-keys)
    - [Natural Idempotency](#activities-natural-idempotency)
  - [Handling Worker Disruptions](#activities-handling-worker-disruptions)
  - [A Grand Unified Theory](#activities-a-grand-unified-theory)
- [Part Two: Workflows](#part-two-workflows)
  - [Living Within Server Limits](#workflows-living-within-server-limits)
  - [Keeping Code Deterministic](#workflows-keeping-code-deterministic)
  - [Versioning Code Changes](#workflows-versioning-code-changes)
    - [Evaluate Change Versions Up Front](#workflows-evaluate-change-versions-up-front)
    - [Verifying Replay Safety](#workflows-verifying-replay-safety)
    - [Cleaning Up Change Versions](#workflows-cleaning-up-change-versions)
  - [Disconnected Child Workflows](#workflows-disconnected-child-workflows)
  - [Use Timers, Not Timeouts](#workflows-use-timers-not-timeouts)
  - [Cap Workflow Lifetime With ContinueAsNew](#workflows-cap-workflow-lifetime-with-continueasnew)
  - [Design Guidelines](#workflows-design-guidelines)
- [Acknowledgements](#acknowledgements)

---

# Part One: Activities

Activities are how Temporal workers interact with the outside world.

Activities must deal with:

- Downstream service errors
- Worker crashes & hangs
- Temporal server disruptions

> [!TIP]
> Activities have a lot of responsibility. They're the main window through which workflows are able to interact with the outside world. That means they also have to put up with all sorts of system disruptions that our workflow code can happily sleep through until it's time to be woken up again.

---

## Temporal's shared responsibility model for activities:

Temporal server provides an "at least once" execution semantic and retries activities by default.

It's on us to:

- Implement idempotency
- Gracefully handle cancelation
- Not amplify bad requests

> [!TIP]
> The biggest way Temporal makes this easier for us is by retrying activities by default. Specifically Temporal gives us an "at least once" execution semantic for every activity.
>
> And that's really convenient, but Temporal can't automatically ensure that our activities don't have undefined behavior if you run them more than once, or that they detect and handle cancelation, or that when there's a downstream service outage our retry policies don't conjure up a storm of attempts that only make matters worse.
>
> So let's dive into how to write reliable, well-behaved activities.

---

## Activities: Handling Downstream Service Errors

```go
func ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    httpReq := newPaymentHTTPReq(req) // POST api.example.com/v1/payments/charge
    // Send request
    resp, err := httpClient.Do(httpReq)
    if err != nil { return nil, err }

    switch resp.StatusCode {
    case http.StatusOK:
        // Decode the response and return
        var cr ChargeResponse
	    err = json.NewDecoder(resp.Body).Decode(&cr)
	    return &cr, err
    default:
        return nil, fmt.Errorf("unexpected http status: %s", resp.StatusCode)
    }
}
```

> [!TIP]
> We'll start out with an example activity that's responsible for charging a customer through a payments API.
>
> And you can see there are already a few places where we might return an error. So one of the first things we need to consider is whether there are types of **failure conditions that _shouldn't_ result in the activity being retried**.

---

## Activities: Avoid Amplifying Invalid Requests

```go
func ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    httpReq := newPaymentHTTPReq(req) // POST api.example.com/v1/payments/charge
    // Send request ...

    switch resp.StatusCode {
    case http.StatusOK: /* Decode the response and return ... */
    case http.StatusBadRequest:
        // Return non-retryable error
        return nil, temporal.NewNonRetryableApplicationError(resp.Status, "http_400", nil)
    default:
        return nil, fmt.Errorf("unexpected http status: %s", resp.StatusCode)
    }
}
```

> [!TIP]
> One of those conditions would be if the payments API responds with a "Bad Request" HTTP status code. Since retrying won't change the shape of the request being sent, we should probably update our activity to **translate this into a non-retryable error** so that the activity fails fast.

---

## Activities: Avoid Overloading Services With Retries

```go
func ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    httpReq := newPaymentHTTPReq(req) // POST api.example.com/v1/payments/charge
    // Send request ...

    switch resp.StatusCode {
    case http.StatusOK: /* Decode the response and return ... */
    case http.StatusBadRequest: /* Return non-retryable error ... */
    case http.StatusTooManyRequests:
        // Honor the server's hint when present; otherwise back off
        // more aggressively than the policy's default.
        delay := activityhelpers.ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
        if delay == 0 {
            delay = activityhelpers.GetNextRetryDelay(ctx) * 2
        }
        return nil, temporal.NewApplicationErrorWithOptions(resp.Status, "http_429",
            temporal.ApplicationErrorOptions{NextRetryDelay: delay})
    default:
        return nil, fmt.Errorf("unexpected http status: %s", resp.StatusCode)
    }
}
```

> [!TIP]
> It's also possible that there is a problem with the downstream API service provider or we are approaching a rate limit. In that case an error might be retryable, but we should **increase our backoff time** before the next retry to avoid making the problem worse.
>
> So now our activity checks for the TooManyRequests HTTP status and calculates a next retry delay based on the `Retry-After` HTTP response header if it has been set. Otherwise we can just increase the next retry delay be a factor of 2.

---

## Activities: Weathering System Outages

```go
func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts:    3,
            InitialInterval:    time.Second,
            BackoffCoefficient: 2,
            MaximumInterval:    100 * time.Second,
        },
    })
    resp, err := workflowhelpers.AwaitActivity(ctx, ChargePayment, chargeRequest)

    // ...
}
```

> [!TIP]
> Switching contexts to the workflow code that is invoking our ChargePayment activity, now we need to think about what timeouts and retry policy makes sense for what the activity does.
>
> In this case we've started with a 30 second Start To Close timeout because we're not doing any computation in the activity and we expect the payments API to respond fairly quickly.
>
> We've also set a 1 minute Schedule To Close timeout, which should be plenty of time to do a few retries if the activity fails quickly.
>
> But we also need to consider the worst case: what if our worker, or the payments API, are down for an extended period of time? Would it be preferable for our workflow to observe that the activity has timed out after 1 minute, or is it better to allow more time for an outage to be resolved and for the activity to complete successfully?

---

## Activities: Weathering System Outages (continued)

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
     // ... generate a charge request for the item & customer

     ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
         StartToCloseTimeout:    30 * time.Second,
-        ScheduleToCloseTimeout: time.Minute,
+        ScheduleToCloseTimeout: time.Hour, // allow retrying for up to 1 hour
         RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts:    3,
            InitialInterval:    time.Second,
            BackoffCoefficient: 2,
            MaximumInterval:    100 * time.Second,
        },
     })

     resp, err := workflowhelpers.AwaitActivity(ctx, ChargePayment, chargeRequest)
     // ...
 }
```

> [!TIP]
> Let's say we want to be able to recover after outages lasting up to about an hour.
>
> We can allow this by updating the ScheduleToClose timeout to an hour, which means there's a much **larger window for incidents to be resolved through activity retries** without the workflow ever straying from its happy path. And because we're categorizing the errors returned from the activity function as retryable vs. not, we also don't need to worry so much about the tradeoff between failing fast and having more time for system recovery.
>
> So now we're ready to go, right?

---

## Activities: Weathering System Outages (continued)

```go
func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour, // allow retrying for up to 1 hour
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts:    3, // oops
            InitialInterval:    time.Second,
            BackoffCoefficient: 2,
            MaximumInterval:    100 * time.Second,
        },
    })

    // ... execute the ChargePayment activity
}
```

> [!TIP]
> Almost!
>
> The last thing to double check is our retry policy. Right now it's configured to only allow a maximum of 3 attempts, which means that it's possible we will exhaust our retries really quickly.

---

## Activities: Weathering System Outages (continued)

![Retry simulator showing max attempts exhausted](../.assets/retry_simulator_max_attempts.png)

[docs.temporal.io/develop/activity-retry-simulator](https://docs.temporal.io/develop/activity-retry-simulator)

> [!TIP]
> Temporal provides a simulator to inspect the behavior of your timeout and retry policy configuration which is helpful since there's some math involved.
>
> By plugging in the policy from the previous slide, we can confirm that the activity may actually be marked as failed after only 6 seconds! This is way off our goal of surviving outages up to 1 hour.

---

## Activities: Weathering System Outages (continued)

```go
func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
        RetryPolicy: &temporal.RetryPolicy{
            // MaximumAttempts: 0, // unlimited; bound by ScheduleToCloseTimeout
            InitialInterval:    time.Second,
            BackoffCoefficient: 2,
            MaximumInterval:    100 * time.Second,
        },
    })

    // ... execute the ChargePayment activity
}
```

> [!TIP]
> Of course we could increase the max attempts in our retry policy, and go back to the simulator to make sure there are enough attempts to reach our desired schedule to close timeout.
>
> But a simpler mental model is to **skip setting maximum attempts** at all, so that you automatically get as many retries as fit within the schedule to close timeout.
>
> Note that you can still use retry policies to fine tune the initial and maximum retry intervals as needed such that the activity doesn't retry too frequently before the timeout is reached.
>
> But for these additional properties of retry policies, Temporal already sets pretty good defaults for most use cases. And those are what we're looking at here: by default, the first retry happens 1 second after the first activity failure, and the interval doubles from there until it caps off at 100 seconds between each attempt.

---

## Activities: Weathering System Outages (continued)

```go
func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
    })

    // ... execute the ChargePayment activity
}
```

> [!TIP]
> So the last tweak we'll make to the activity options is simply to remove the retry policy, and use the Temporal default of unlimited attempts and exponential backoff.

> [!NOTE]
> It is not always the case that you should avoid max attempts or that it is necessary to always have a long schedule to close timeout. But it is important to **have a target in mind for how long your activity should be capable of retrying during a disruption**, and to **verify that your retry policy meets that goal**.

---

## Activities: Implementing Idempotency

> "Idempotence is the property of certain operations in mathematics and computer science whereby they can be applied multiple times without changing the result beyond the initial application."

[wikipedia.org/wiki/Idempotence](https://en.wikipedia.org/wiki/Idempotence)

> [!TIP]
> A term that gets thrown around a lot when talking about activities is idempotency. This just means that **if you run the activity more than once with the same input, you should get the same result**.
>
> It turns out this is a really important property for activities to have, because they're getting retried all the time. And we really don't want to do something like charging a customer 20 times for the same purchase just because there was a temporary system outage.

---

## Activities: Implementing Idempotency

Common techniques to achieve idempotency:
- Passing idempotency key to external APIs
    - Derive a key from the Workflow ID and activity ID. Pass this to downstream systems (like Stripe) to ignore duplicate requests.
- Applying database constraints
    - Use `INSERT ... ON CONFLICT` or conditional writes to ensure records aren't created twice.
- Using naturally idempotent operations
    - Design side effects as state settings (Set to X) rather than increments (+1), or use upserts with fixed IDs.
    - May help to decompose into multiple activities.

> [!TIP]
> Implementing and testing for idempotency is still not a solved problem. But there are some common techniques, and if you're lucky your activities are interacting with external services which themselves are designed for idempotency.

---

## Activities: Implementing Idempotency

Caution: Don't rely on `MaxAttempts: 1` in retry policy!

> Note that **idempotent behavior is still required** even if you set MaxAttempts to 1 in your retry policy. Temporal does not guarantee at most once execution for activities; this has to do with the way server replication and failover works.

---

## Activities: Idempotency Keys

```go
func ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    httpReq := newPaymentHTTPReq(req) // POST api.example.com/v1/payments/charge
    httpReq.Header.Set("Idempotency-Key", getIdempotencyToken(ctx))
    // Send request ...

    switch resp.StatusCode {
    case http.StatusOK: /* Decode the response and return ... */
    case http.StatusBadRequest: /* Return non-retryable error ... */
    case http.StatusTooManyRequests: /* Back off more aggressively ... */
    default: /* Unexpected status ... */
    }
}

func getIdempotencyToken(ctx context.Context) string {
    i := activity.GetInfo(ctx)
    key := fmt.Sprintf("%s:%s:%s",
        i.WorkflowExecution.ID,
        i.WorkflowExecution.RunID,
        i.ActivityID,
    )
    return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
```

> In our payment example from earlier, the activity is calling an API that supports the `Idempotency-Key` HTTP header.
>
> So here we can add a function to **generate an opaque string based on the workflow run ID and activity ID**, which will be the same across every activity attempt while still being unique in case the same workflow scheduled multiple payment activities.
>
> And then we can just pass that along to the payments API!

---

## Activities: Natural Idempotency

```go
func RunKubernetesJob(ctx context.Context, req RunJobRequest) (*RunJobResponse, error) {
    jobs := k8sClient.BatchV1().Jobs(req.Namespace)
    // Create the job
    _, err := jobs.Create(ctx, &batchv1.Job{ /* ... */ })
    if err != nil {
        return nil, err
    }

    // Poll for final status
    for {
        activity.RecordHeartbeat(ctx)
        j, err := jobs.Get(ctx, req.Name)
        if err != nil {
            return nil, err
        }
        if status := getJobStatus(j); status.IsTerminal() {
            return &RunJobResponse{Status: status}, nil
        }
        time.Sleep(15 * time.Second)
    }
}
```

> In the real world, we also tend to have activities that perform multiple operations and interact with systems that don't have nice primitives for idempotency. Like in this new example, where our activity is responsible for creating a Kubernetes job, and then waiting for it to complete before reporting the final job status.
>
> The problem here is that if the activity fails or the worker is redeployed during the polling loop, then on the next attempt the activity will fail at creating a job with the same name instead of skipping creation and getting the status of the existing job. **No matter how many times the activity is retried, it would keep hitting that same error**.

---

## Activities: Natural Idempotency (continued)

```diff
 func RunKubernetesJob(ctx context.Context, req RunJobRequest) (*RunJobResponse, error) {
     jobs := k8sClient.BatchV1().Jobs(req.Namespace)
     // Create the job if it doesn't exist
     _, err := jobs.Create(ctx, &batchv1.Job{ /* ... */ })
-    if err != nil {
+    if err != nil && !apierrors.IsAlreadyExists(err) {
         return nil, err
     }
 
     for { /* Poll for final status... */ }
 }
```

> The smallest fix is to add an already exists check to the job create step in the activity. I left it out here, but we'd probably also want to verify that the existing job spec matches our expected spec and replace it or error out not.
>
> So it's possible to keep activities idempotent, even if they perform multiple operations.

---

## Activities: Natural Idempotency (continued)
<!-- Update code example: Split into two activities, one called StartKubernetesJob and another called AwaitKubernetesJob. -->
```go
func StartKubernetesJob(ctx context.Context, req StartJobRequest) error {
    jobs := k8sClient.BatchV1().Jobs(req.Namespace)
    // Create the job if it doesn't exist
    _, err := jobs.Create(ctx, &batchv1.Job{ /* ... */ })
    if err != nil && !apierrors.IsAlreadyExists(err) {
        return err
    }
    return nil
}

func AwaitKubernetesJob(ctx context.Context, req AwaitJobRequest) (*AwaitJobResponse, error) {
    for { /* Poll for final status... */ }
}
```

> But in this case, splitting responsibilities across two separate activities is probably an even better approach, because now we can set independent timeouts and retry policies for each step of a) creating the job, and b) waiting for it to complete. We also automatically get more visibility into what's going on in the workflow history.
>
> Note that there is still an already exists check in the new `StartKubernetesJob` activity. Even though it may not be likely, **a network partition or a poorly timed worker crash could still mean that the activity is retried even after the Create API call succeeds**. But reducing the scope of what the activity does still makes reasoning about idempotency a bit simpler.

---

## Activities: Handling Worker Disruptions

```go
func RunKubernetesJob(ctx workflow.Context, req RunJobRequest) (*RunJobResponse, error) {
    // Execute the StartKubernetesJob activity
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            StartToCloseTimeout:    time.Hour,
            ScheduleToCloseTimeout: time.Hour,
        }),
        AwaitKubernetesJob,
        AwaitJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

> In the workflow that invokes our activities to create and wait for completion of the Kubernetes job, again we need to choose some timeout values. Here we started out with a 1 hour StartToClose and ScheduleToClose timeout because we wanted to allow the Kubernetes job to run for up to an hour, and the activity itself shouldn't be timed out as long as it's still polling the job status in that for loop.
>
> The only problem here is that **if the worker crashes or terminates unexpectedly, then it will never report a final result or error for the activity**. And since the ScheduleToClose timeout is the same duration as StartToClose timeout, **Temporal won't retry the activity on our behalf** either.
>
> We could decrease the StartToClose timeout to allow the activity to be retried if the worker fails. But that forces a tradeoff between how quickly Temporal retries after a worker failure and how often we are doing unnecessary retries (which in some cases, could mean repeating expensive work).

---

## Activities: Handling Worker Disruptions (continued)

<!-- Updated code example: Replace the start to close timeout with a 30s heartbeat timeout. -->
```go
func RunKubernetesJob(ctx workflow.Context, req RunJobRequest) (*RunJobResponse, error) {
    // Execute the StartKubernetesJob activity
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            // Long-running activity: prefer Heartbeat over StartToClose timeout.
            HeartbeatTimeout:       30 * time.Second,
            ScheduleToCloseTimeout: time.Hour,
        }),
        AwaitKubernetesJob,
        AwaitJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

> A more elegant solution is to swap out the StartToClose timeout for a Heartbeat timeout.
>
> This allows our activity to run for as long as it needs to, up to the ScheduleToClose timeout, without being stopped and retried, as long as it keeps reporting back to the Temporal server via heartbeat messages. If the worker fails to send heartbeats, then Temporal will retry the activity.
>
> When choosing a value for Heartbeat or StartToClose timeout, the question to answer is **how long you want to wait for the activity's retry policy to kick in if the worker fails**. A good starting point is 30 seconds, but you may want to go higher in order to reduce the cost of heartbeats for long-running activities.
>
> Note that **sending heartbeats is also the only way for the activity to detect that it was canceled**. Heartbeats are therefore required for any activity that needs to be stopped or that should perform some cleanup logic if it is explicitly canceled by the workflow or the workflow completes before the activity.

---

## Activities: A Grand Unified Theory

1. Activities should be idempotent.
2. Always set a **ScheduleToClose** timeout. Base the value on how long the activity should continue retrying during a worst case outage.
3. Always set either **Heartbeat** or **StartToClose** timeout. Use **StartToClose** timeout only when the activity is guaranteed to not run past that duration and it is acceptable to wait the whole duration before a retry. 
4. Activities that perform cleanup on cancelation MUST send heartbeats.
5. Prefer unlimited attempts with **ScheduleToClose** as the bound.
6. Respect error conventions from downstream services. Translate these into Temporal application errors to skip retry or adjust backoff behavior.

_Consider implementing and/or enforcing these policies in an interceptor._

---

# Part Two: Workflows

Need to be robust to:
- Server-imposed history & payload limits
- Workflow code changing across deployments
- Signals arriving in unpredictable order
- Cancelation requests at any point in execution

Must also:
- Stay deterministic across replays
- Yield quickly to the workflow task event loop

> Workflows definitely come with their own challenges. We're going to cover a few of these, including runtime limitations, determinism, and versioning code changes.

---

## Workflows: Living Within Server Limits

Server-imposed limits to be aware of:
- **Individual payload size**: ~2MB per workflow/activity input or output, signal, or update.
- **Workflow history bytes**: 50MB (<10MB recommended). Results in termination.
- **Workflow history length**: 50k events (<10k recommended). Results in termination.
- **Workflow task timeout**: 10s default (120s max configurable). Results in failed workflow task (retried).
- **Workflow lock contention**: No hard limit, but aim for no more than ~1 workflow state change per second.

Mitigations:
- Use **ContinueAsNew** to start a fresh execution with reset history. Check `GetContinueAsNewSuggested()` to know when the server is recommending it.
- Avoid passing large payloads from activities to workflows, or use [external payload storage](https://docs.temporal.io/external-storage).

> There are several dimensions in which workflows are constrained at runtime.
>
> The first is that individual payload sizes peristed in workflow history can't go past around 2MB. This is a limitation that is inherited from the Temporal gRPC API.
>
> Limits on the workflow history include total length, which is recommended to stay under 10,000 events, and the total workflow history size should be less than 10 MB. These limits exist to ensure that workflow histories can be replayed quickly whenever the workflow needs to be loaded into memory on a worker.
>
> Temporal also enforces a workflow task timeout of 10 seconds by default, which is how long the workflow function has to return the next command. This should almost never need to be changed unless you are using external payload storage.
>
> And the last limitation is workflow lock contention, which you can run into if your workflow deals with a high throughput of incoming signals or executes activities with high parallelism.
>
> Typically if you're approaching one of these limits, it means you need to start using `ContinueAsNew` or enable external payload storage, or branch out work using child workflows.

---

## Workflows: Keeping Code Deterministic

Common sources of non-determinism in workflow code:
- Network calls (HTTP, DB queries, gRPC) -- re-execute on every replay and may return different results
- System time (`time.Now()` vs `workflow.Now()`)
- Random number generation
- Usage of environment variables
- Interactions with the filesystem
- Goroutines spawned outside `workflow.Go` -- the SDK doesn't record their scheduling, and they often race with the workflow function for shared state
- Variable references from outside the workflow function scope

> Another thing to be aware of is that workflow code must be deterministic. Temporal uses event sourcing under the hood to be able to recreate the state of workflows in your worker's memory on demand, so it's very important that workflow functions always produce the same state when replaying workflow histories.

---

## Workflows: Keeping Code Deterministic (continued)

Tools to catch non-determinism:
- **Go**: [`workflowcheck`](https://github.com/temporalio/sdk-go/tree/master/contrib/tools/workflowcheck) -- opt-in static analyzer that flags non-deterministic calls in workflow code.
- **Python**: [Workflow sandbox](https://docs.temporal.io/develop/python/python-sdk-sandbox) -- enabled by default; restricts imports and module access at runtime.
- **TypeScript**: V8 isolate sandboxing is built-in -- workflow code runs in a separate V8 context with no Node.js APIs.

> The Temporal team has done a great job making determinsm easier to implement by providing static analysis tools and by deeply integrating with language runtimes. I recommend checking out what tooling is available for your language, and be careful not to let a coding assistant run rampant adding exceptions to determinism rules just because it deems them pesky.

---

## Workflows: Versioning Code Changes

Change versioning is used to gate new behavior in workflow functions in order to maintain backwards compatability with workflows started on earlier versions of the worker.

Change version lifecycle:
1. Add new behavior, gated with version check
2. Wait for workflows started on previous version of the worker to complete
3. Remove the old behavior and the version check

> Another really common challenge with workflows is shipping new versions of the code. Almost any new behavior added to an existing workflow function needs to be gated with a change version check -- this is called a patch in most SDKs.
>
> But beyond just making sure we use change versions when modifying workflows, we should also be cleaning up change versions in our codebase so that all of those logic branches don't accrue over time.

---

## Workflows: Versioning Code Changes

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
     // ...

     sel.AddFuture(workflow.NewTimer(ctx, 12*time.Hour), func(f workflow.Future) {
         err = workflow.ErrDeadlineExceeded
     })

     sel.Select(ctx)
     if err != nil {
+        delayVersion := workflow.GetVersion(ctx, "handle-shipment-delay", workflow.DefaultVersion, 1)
+        switch delayVersion {
+        case 1:
+            workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
+        }
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }
```

> So let's look at an example workflow code change. We're back in the PurchaseItem workflow, and the goal is to add some logic that executes a refund child workflow if the shipment isn't received on time.
>
> This is a well formed change version check, and deploying as-is would not cause immediate problems. But there are two subtle issues at play.
>
> First, the change version is being evaluated inside of a conditional branch means that not all workflow executions will actually evaluate it. Right now, only workflows that don't receive the shipment processed signal within 12 hours will register the change version.
>
> Second, the change version is being evaluated late in the workflow's execution; in this case we know that the workflow could be running for at least 12 hours before it registers the change version.
>
> The reason this matters is that we want ALL workflow executions started after the new version of the worker is deployed to register the same set of change versions so that we can automate checks to verify that removal of the change version is safe in the future.

---

## Workflows: Evaluate Change Versions Up Front

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
+    delayVersion := workflow.GetVersion(ctx, "handle-shipment-delay", workflow.DefaultVersion, 1)
     // ...

     sel.AddFuture(workflow.NewTimer(ctx, 12*time.Hour), func(f workflow.Future) {
         err = workflow.ErrDeadlineExceeded
     })

     sel.Select(ctx)
     if err != nil {
+        switch delayVersion {
+        case 1:
+            workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
+        }
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }
```

> The fix is simple, we just need to hoist the version check to the top of the workflow so every execution records the version as soon as it starts, even if the code branch it's used in is never evaluated.

---

## Workflows: Evaluate Change Versions Up Front (continued)

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
-    delayVersion := workflow.GetVersion(ctx, "handle-shipment-delay", workflow.DefaultVersion, 1)
+    delayVersion := workflow.GetVersion(ctx, "handle-shipment-delay", workflow.DefaultVersion, 2)
     // ...

     sel.Select(ctx)
     if err != nil {
         switch delayVersion {
         case 1:
             workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
+        case 2:
+            // Cancel the in-flight shipment before issuing the refund.
+            if e := workflowhelpers.AwaitActivity(ctx, CancelShipment, cancelRequest); e != nil {
+                log.Warn("failed to cancel shipment", "error", e)
+            }
+            workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
         }
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }
```

> And if we ever need to make more updates to that area of the workflow in the future, then we can bump the change id's max version again and add another branch with our new logic.
>
> In our example the payment workflow now now cancels the in-flight shipment before issuing the refund so the package doesn't get shipped after the customer has been refunded.
>
> When reviewed in isolation, we can often be pretty confident about a change like this, but in order to scale verification of replay safety to larger and more frequent changes, we can get some help from our friends, the computers.

---

## Workflows: Verifying Replay Safety

```bash
# Find the earliest workflow that did not hit either patch branch
LIST_QUERY="WorkflowType='PurchaseItem'"
LIST_QUERY+=" AND TemporalChangeVersion NOT IN ('handle-shipment-delay-1', 'handle-shipment-delay-2')"
temporal workflow list --query "$LIST_QUERY" --order-by 'StartTime ASC' --limit 1

# Find the earliest workflow that took version 1 of the patch.
LIST_QUERY="WorkflowType='PurchaseItem'"
LIST_QUERY+=" AND TemporalChangeVersion IN ('handle-shipment-delay-1')"
temporal workflow list --query "$LIST_QUERY" --order-by 'StartTime ASC' --limit 1

# Download workflow history to a test fixture path.
temporal workflow show --workflow-id <ID> --output json \
  > testdata/purchase_item_history_<PATCH_VERSION>.json
```

> Temporal provides a harness to replay any workflow history against your workflow function locally, without ever deploying a worker. So in order to verify a change is replay safe programatically, we just need to find a workflow history to test against.
>
> What I'm showing here is a quick way to find the earliest workflow execution that ran on the version of our code that didn't have the handle-shipment-delay change version, and a second command to find the earliest workflow that took the version 1 branch. The query is using the TemporalChangeVersion search attribute that is automatically registered when you evaluate a change version in a workflow.
>
> And then in the third command, we're grabbing the workflow history and saving to a JSON file to replay in a unit test.

---

## Workflows: Verifying Replay Safety (continued)

```go
func TestReplayWorkflowHistory(t *testing.T) {
    testhelpers.AssertWorkflowReplayFromJSONFiles(t, 
        PurchaseItem,
        "testdata/purchase_item_history_v0.json",
        "testdata/purchase_item_history_v1.json",
    )
}
```

**Check the code coverage for the version branches in your workflow** -- if they aren't covered, the replay test is not validating your change.

Find more techniques at [temporal.io/resources/on-demand/replay-safety-at-datadog](https://temporal.io/resources/on-demand/replay-safety-at-datadog)

> And this is what that unit test might look like. We can run this locally or in CI, and if the new code's command sequence diverges from either workflow history, then the test fails and we can be alerted before shipping the new workflow code to production.
>
> I definitely recommend setting up a test harness for workflow replay, because it can also be a super powerful way to debug workflows locally when things go wrong.
>
> Having a replay test harness also makes it pretty easy to compute code coverage for your workflow and visualize it in an IDE just by running that specific test; you should be doing this to ensure that the replay test is actually exercising the parts of the workflow function that you updated.
>
> If you're interested in more techniques to ensure replay safety for workflow code changes, then please check out the talk that my colleague Jing Yi gave a couple years ago titled Replay Safety at Datadog.

---

## Workflows: Cleaning Up Change Versions

```bash
#!/bin/sh
# Prints 0 when no in-flight workflows are still on v0 or v1.

# Timestamp for 5 minutes ago, to avoid counting workflows that have been
# scheduled but haven't yet been picked up by a worker. (use `date` on Linux)
CUTOFF=$(gdate -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S.%3NZ)

# Count workflows still running on a handle-shipment-delay version < 2.
QUERY="WorkflowType='PurchaseItem'"
QUERY+=" AND ExecutionStatus='Running'"
QUERY+=" AND TemporalChangeVersion NOT IN ('handle-shipment-delay-2')"
QUERY+=" AND StartTime < '$CUTOFF'"

temporal workflow count --query "$QUERY"
```

> Now after we've deployed a workflow code change, it's time to wait for the old workflows to complete and then clean up the change version branches.
>
> To verify this is safe, we can run a query to check if there are still any workflows running that were started with an earlier change version.
>
> Note that you'd need to do this once per namespace or Temporal cluster if you have multiple deployments of your worker.

---

## Workflows: Cleaning Up Change Versions (continued)

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
-    delayVersion := workflow.GetVersion(ctx, "handle-shipment-delay", workflow.DefaultVersion, 2)
+    // TODO: drop this once the new worker is rolled out everywhere.
+    _ = workflow.GetVersion(ctx, "handle-shipment-delay", 2, 2)
     // ...

     sel.Select(ctx)
     if err != nil {
-        switch delayVersion {
-        case 1: /* Refund only ... */
-        case 2:
         if e := workflowhelpers.AwaitActivity(ctx, CancelShipment, cancelRequest); e != nil {
             log.Warn("failed to cancel shipment", "error", e)
         }
         workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }
```

> After we've confirmed there are no workflows running with earlier change versions, we can clean up the old code branches.
>
> Note that it is HIGHLY recommended to do this in two phases; first bumping the min supported version to match the max version like we're doing here, and a later deployment to remove the change version entirely.
>
> The reason to continue evaluating the change version for one additional deploy cycle is that while your rollout progresses, it's possible for a workflow execution to bounce between the current and previous versions of your worker. So that change version marker is still needed in order to avoid the previous version of the worker going down the wrong path if it evaluates a workflow that was started on the current version of the worker.

---

## Workflows: Disconnected Child Workflows

```go
func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
    // ...
    sel.Select(ctx)
    if err != nil {
        if e := workflowhelpers.AwaitActivity(ctx, CancelShipment, cancelRequest); e != nil {
            log.Warn("failed to cancel shipment", "error", e)
        }
        workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
        return nil, err
    }
    return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
}
```

> An external fulfillment system signals the workflow once the shipment is processed. A Selector fans in that signal, a fulfilment deadline, and ctx cancelation; whichever fires sets err. On err, the workflow tries to refund via a child workflow.
>
> But this naive version uses the parent's (possibly canceled) ctx, the default ParentClosePolicy, and doesn't wait for the child to be scheduled before returning. Each of those is a bug we'll fix on the next slide.
>
> There is another subtle issue with the workflow code we've just been looking at.
>
> When we execute the RefundPayment child workflow, we're not actually waiting for it to complete before returning from our workflow function.
>
> This is what was intended; the Purchase workflow is designed to return as soon as it detects the shipment error, while the refund child workflow is meant to complete asynchronously.

---

## Workflows: Disconnected Child Workflows (continued)

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
     // ...
     sel.Select(ctx)
     if err != nil {
         if e := workflowhelpers.AwaitActivity(ctx, CancelShipment, cancelRequest); e != nil {
             log.Warn("failed to cancel shipment", "error", e)
         }
-        workflow.ExecuteChildWorkflow(ctx, RefundPayment, refundRequest)
+        execDisconnectedChildWorkflow(ctx, RefundPayment, refundRequest)
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }

 func execDisconnectedChildWorkflow(ctx workflow.Context, childWorkflow any, args ...any) {
     ctx = workflow.WithParentClosePolicy(ctx, enums.PARENT_CLOSE_POLICY_ABANDON)
     workflow.ExecuteChildWorkflow(ctx, childWorkflow, args...)
 }
```

> But when a parent workflow returns, by default Temporal will terminate any of its child workflows that are still running. We can change this behavior by setting a parent close policy when starting the child workflow.

---

## Workflows: Disconnected Child Workflows (continued)

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
     // ...
     sel.Select(ctx)
     if err != nil {
         if e := workflowhelpers.AwaitActivity(ctx, CancelShipment, cancelRequest); e != nil {
             log.Warn("failed to cancel shipment", "error", e)
         }
         execDisconnectedChildWorkflow(ctx, RefundPayment, refundRequest)
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }

 func execDisconnectedChildWorkflow(ctx workflow.Context, childWorkflow any, args ...any) {
     ctx = workflow.WithParentClosePolicy(ctx, enums.PARENT_CLOSE_POLICY_ABANDON)
+    ctx, _ = workflow.NewDisconnectedContext(ctx)
     workflow.ExecuteChildWorkflow(ctx, childWorkflow, args...)
 }
```

> It is also possible that at this point the Purchase workflow has been canceled. Even with a custom parent close policy, attempting to start the refund workflow would fail in this case. So we also need to be explicit that the child workflow should be started with a disconnected context that is unaffected by cancelation.

---

## Workflows: Disconnected Child Workflows (continued)

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
     // ...
     sel.Select(ctx)
     if err != nil {
         if e := workflowhelpers.AwaitActivity(ctx, CancelShipment, cancelRequest); e != nil {
             log.Warn("failed to cancel shipment", "error", e)
         }
         execDisconnectedChildWorkflow(ctx, RefundPayment, refundRequest)
         return nil, err
     }
     return &PurchaseResponse{TrackingID: shipmentResponse.TrackingID}, nil
 }

 func execDisconnectedChildWorkflow(ctx workflow.Context, childWorkflow any, args ...any) {
     ctx = workflow.WithParentClosePolicy(ctx, enums.PARENT_CLOSE_POLICY_ABANDON)
     ctx, _ = workflow.NewDisconnectedContext(ctx)
-    workflow.ExecuteChildWorkflow(ctx, childWorkflow, args...)
+    fut := workflow.ExecuteChildWorkflow(ctx, childWorkflow, args...)
+    if err := fut.GetChildWorkflowExecution().Get(ctx, nil); err != nil { panic(err) }
 }
```

> And the last issue here is especially subtle: if we only "start" the child workflow and immediately return, it may not actually be started. So before returning we also need to get the child workflow execution future to ensure it was created by Temporal. This is a Go SDK specific issue.

---

## Workflows: Use Timers, Not Timeouts

```diff
 func PurchaseItem(ctx workflow.Context, req PurchaseRequest) (*PurchaseResponse, error) {
     // ...

+    // Reserve at least 1 minute for compensation before the hard timeout.
+    sel.AddFuture(getSoftTimeout(ctx, time.Minute), func(f workflow.Future) {
+        // Run compensating actions now! Workflow terminating in 1 minute...
+    })

     // ...
 }

 func getSoftTimeout(ctx workflow.Context, padding time.Duration) workflow.Future {
     timeout := workflow.GetInfo(ctx).WorkflowRunTimeout
     if timeout <= padding {
         panic("WorkflowRunTimeout too short")
     }
     return workflow.NewTimer(ctx, timeout - padding)
 }
```

> In any workflow that needs to perform compensating actions, like our purchase example that refunds customers when shipment fails, it is also important to ensure that the workflow timeout set by the client when starting the workflow cannot elapse before the compensating action has been executed.
>
> When a workflow execution times out, the result is functionally the same as workflow termination. If you need a chance to perform compensating actions, create a deadline from inside the workflow using a timer and verify that the timer will fire before the actual workflow run timeout is reached.

---

## Workflows: Cap Workflow Lifetime With ContinueAsNew

```diff
 func SubscriptionWorkflow(ctx workflow.Context, state SubscriptionState) error {
     // ...
+    sel.AddFuture(workflow.NewTimer(ctx, 24*time.Hour), func(f workflow.Future) {
+        // Noop; just unblock the selector
+    })

     for sel.HasPending() {
         sel.Select(ctx)
+        if continueAsNewSuggested(ctx) {
+            return workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, state)
+        }
         // ...
     }
 }

 func continueAsNewSuggested(ctx workflow.Context) bool {
     info := workflow.GetInfo(ctx)
     return info.GetContinueAsNewSuggested() ||
         workflow.Now(ctx).After(info.WorkflowStartTime + 24*time.Hour - time.Second)
 }
```

> Another way timers can be handy is when dealing with long-running workflows. I think of it as a general best practice to implement ContinueAsNew for any workflow which may run for longer than 24 hours, and to trigger ContinueAsNew proactively based on a timer even if no other state transitions are happening in the workflow. If you deploy once per day, then following this advice means you can safely add, deprecate, and remove any change version within one week.
>
> And as an added bonus, if you onboard to worker versioning then time based continue as new will also ensure that old worker versions do not need to remain active for longer than one day.

---

## Workflows: Design Guidelines

Workflow functions:
- MUST be deterministic. Use the Temporal SDK for time, randomness, and side effects.
- MUST evaluate all change versions as the first step (at the top of the function).
- SHOULD use internal timers rather than execution timeouts if they need to run compensating actions.
- SHOULD be designed to complete or ContinueAsNew within 24 hours or when the server suggests ContinueAsNew.
- SHOULD drain all signals before completing or ContinueAsNew.
- SHOULD fan out large batches of work to child workflows.

Temporal workers:
- MUST be configured with a BuildID
- SHOULD have a replay testing harness.
- SHOULD be onboarded to worker versioning and pinned workflows.
- SHOULD enable external payload storage if activity responses are ~1MB or larger.

---

# Acknowledgements

- [100 Go Mistakes](https://100go.co) by Teiva Harsanyi
- Loïc Minaudier (software engineer @Datadog)
- The hundreds of engineers using Temporal at Datadog
- Participants in the "Temporal Pain Points" birds of a feather (Replay 2025)

<!-- BoaF notes: https://docs.google.com/document/d/1lfITwMgWT3eyY7qpd64YKHmjtHL4KhFonZqYawJsEdM/edit -->
