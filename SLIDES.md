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

<!-- TODO: Make this sentence sound less smarmy. -->
A field guide to the bugs you'll write before you write them.

A working catalog of failure modes drawn from production Temporal codebases — what breaks, why it breaks, and the smallest change that makes it stop.

What we'll cover
- Activities: Errors, retries, idempotency, timeouts, worker disruption
- Workflows: Limitations, determinism, change versioning
- Recommendations to simplify working with Temporal day to day

Follow Along:
<!-- QR code linking to jacob.work/100TM (slides in markdown format for those who want to follow along with code examples) -->

<!-- Speaker notes:
A few years ago I got looped into a project at work to help a team launch a new product called Datadog Oncall (and I promise this is not an ad). But the reason I was looped in was because they were planning to build it on top of Temporal. At Datadog we always want to maintain a high standard of availability and so on for our services, but this had an even higher bar to meet than usual because we wanted be confident in allowing any core engineering team at Datadog to be able to route their own pages through this system despite the potential circular runtime dependencies that you can imagine might make life difficult.

Now I've always enjoyed thinking about all the things that can theoretically go wrong in distributed systems. But suddenly I was fielding all kinds of questions from this new product team about activity execution semantics and retry policies and change versioning and parent close policies and so on, because the team was being so incredibly thorough. And as we were having these conversations, I was wishing that I had some way of capturing these tidbits in a way that could be digestible and simple to follow for anyone else using Temporal at our company.

So that is how 100 Temporal Mistakes was born, and my hope in preparing this talk is that I could shed light on some fo the less obvious things that can go wrong, and also provide practical guidance that you can apply every day when developing Temporal backed applications.

Please note that the advice I'm giving is extremely picky. You certainly don't need to follow all of it, and some may not make sense at all depending on how you're using Temporal. That said, please feel free to roast me in the Q&A if you disagree with anything I say.

Also by the way for anyone who hasn't done the math yet, 100 mistakes in 35 minutes gives us about 20 seconds per mistake. So I'm just going to do a highlights tour, but you can find more content at the link on the slide.
 -->

---

# Part One: Activities

Activities are how Temporal workers interact with the outside world.

Activities have to deal with:
- Downstream service errors
- Worker crashes & hangs
- Temporal server disruptions

<!-- Part One is all about activities. And I'm starting here because, let's face it, workflows are a bit more glamorous with their determinism and durability and signals and so on, but I think there's a lot of subtlety about how we need to design activities and the policy around them that is often glossed over when starting out with Temporal.

So activities have a lot of responsibility. They're the main window through which workflows are able to interact with the outside world. That means they also have to put up with all sorts of system disruptions that our workflow code can happily sleep through until it's time to be woken up again. -->

---

## Temporal's shared responsibility model for activities:

Temporal server provides an "at least once" execution semantic and retries activities by default.

It's on us to:
- Implement idempotency
- Gracefully handle cancelation
- Not amplify bad requests

<!--
The biggest way Temporal makes this easier for us, is by retrying activities by default. Specifically Temporal gives us an "at least once" execution semantic for every activity.

And that's really convenient, but Temporal can't magically ensure that our activities don't have undefined behavior if you run them more than once, or that they detect and handle cancelation, or that when there's a downstream service outage our retry policies don't conjure up a storm of execution attempts that only make matters worse.

So let's dive into how to write reliable, well-behaved activities.
-->

---

## Activities: Handling Downstream Service Errors

```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    httpReq := newPaymentHTTPReq(req) // POST api.example.com/v1/payments/charge

    resp, err := w.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

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

<!-- We'll start out with a common example; this is an activity that's responsible for charging a customer through a payments API.

And you can see there are already a few places where we might return an error. So one of the first things we need to consider is whether there are types of failure conditions that shouldn't result in the activity being retried.
-->

---

## Activities: Avoid Amplifying Invalid Requests

```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    // Send the request ...

    switch resp.StatusCode {
    case http.StatusOK: /* ... */
    case http.StatusBadRequest:
        return nil, temporal.NewNonRetryableApplicationError(resp.Status, "http_400", nil)
    default:
        return nil, fmt.Errorf("unexpected http status: %s", resp.StatusCode)
    }
}
```

<!--
One of those conditions would be if the payments API responds with a "Bad Request" HTTP status code. Assuming that retrying won't change the shape of the request being sent, we should probably update our activity to translate this into a non-retryable error so that the activity fails fast.
-->

---

## Activities: Avoid Overloading Services With Retries

<!-- Updated code example for HTTP 429: prefer the server's Retry-After hint (RFC 7231 §7.1.3 -- delta-seconds or HTTP-date) when present, and fall back to activityhelpers.GetNextRetryDelay with a minimum backoff coefficient of 3 so retries against the rate-limited endpoint back off more aggressively than the workflow's default policy. -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    // Send the request ...

    switch resp.StatusCode {
    case http.StatusOK: /* ... */
    case http.StatusBadRequest: /* ... */
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

<!-- It's also possible that there is a problem with the downstream API service provider or we are approaching a rate limit. In that case an error might be retryable, but we should increase our backoff time before the next retry to avoid making the problem worse.

So now our activity checks for the TooManyRequests HTTP status and calculates a next retry delay based on the `Retry-After` HTTP response header if it has been set. Otherwise we can just increase the next retry delay be a factor of 2.
-->

---

## Activities: Weathering System Outages

<!-- New code example, this time showing the workflow code that invokes the payment activity. Set a 30s start to close timeout and a 1m schedule to close timeout. -->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Minute,
        RetryPolicy: { /* ... */ },
    })
    resp, err := workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)

    // ...
}
```

<!-- Switching contexts to the workflow code that is invoking our ChargePayment activity, we need to think about what timeouts and retry policy makes sense for what the activity does.

In this case we've started with a 30 second Start To Close timeout because we're not doing any computation in the activity and we expect the payments API to respond fairly quickly.

We've also set a 1 minute Schedule To Close timeout, which should be plenty of time to do a few retries if the activity fails quickly.

But we also need to consider the worst case: what if our worker, or the payments API, are down for an extended period of time? Would it be preferable for our workflow to observe that the activity has timed out after 1 minute, or is it better to allow more time for an outage to be resolved and for the activity to complete successfully?
-->

<!-- Speaker note: Temporal is great at retrying activities, but it's still important to think carefully about how we configure timeouts and retry policies in order to survive worst case system outages. For example here I'm invoking my activity with a schedule to close timeout that doesn't give much room for the activity to be retried if the worker or downstream API are temporarily unavailble. -->

---

## Activities: Weathering System Outages (continued)

<!-- Update the schedule to close timeout to 1h in the code example. Include code comment saying "allow retrying for up to 1 hour". -->

<!-- TODO: make this a diff against the previous slide -->
```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour, // allow retrying for up to 1 hour
        RetryPolicy: { /* ... */ },
    })

    resp, err := workflowhelpers.AwaitActivity(ctx, w.ChargePayment, chargeRequest)
    // ...
}
```

<!-- In this case, let's say we want to be able to recover after outages lasting up to about an hour.

We can allow this by updating the ScheduleToClose timeout to an hour, which means there's a much larger window for incidents to be resolved through activity retries without the workflow ever straying from its happy path. And because we're categorizing the errors returned from the activity function as retryable vs. not, we also don't need to worry so much about the tradeoff between failing fast and having more time for system recovery.

So now we're ready to go, right?
-->

<!-- Speaker note: ScheduleToClose doesn't always need to be large. Long values (hours) make sense when the workflow should weather an extended outage and the caller is OK waiting, or is notified asynchronously. Short values (seconds to minutes) make sense when the workflow has a graceful degradation path, when reporting an error quickly is preferable to retrying through an outage, or when an upstream caller is waiting synchronously.
-->

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
            MaximumInterval:    100 * time.Second,
        },
    })

    // ... execute the ChargePayment activity
}
```

<!-- Well, sort of. It turns out we were really thorough and also specified a retry policy. The problem here is that because we only allow a maximum of 3 attempts, in practice we exhaust our retries really quickly. -->

---

## Activities: Weathering System Outages (continued)

<!-- TODO: Inline image src/.assets/retry_simulator_max_attempts.png -->

[docs.temporal.io/develop/activity-retry-simulator](https://docs.temporal.io/develop/activity-retry-simulator)

<!-- To inpect the behavior of your timeout and retry policy config, Temporal actually provides a nice little simulator which is helpful since there's some math involved.

If we plug in the policy from the previous slide, we can see that the activity could actually be marked as failed after only 6 seconds! Obviously this is way off our target of surviving outages up to 1 hour. -->

---

## Activities: Weathering System Outages (continued)

<!-- Comment out the MaxAttempts field in the code example. Include code comment saying "allow unlimited attempts until the ScheduleToClose timeout is reached". -->
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
            MaximumInterval:    100 * time.Second,
        },
    })

    // ... execute the ChargePayment activity
}
```

<!--
Of course we could increase the max attempts in our retry policy, and go back to the simulator to make sure there are enough attempts to reach our schedule to close timeout.

But I think a much simpler mental model is to skip setting maximum attempts at all, so that you automatically get as many retries as fit within your schedule to close timeout.

Note that if you need to you can still use retry policies to fine tune the initial and maximum retry intervals as needed such that you don't retry too frequently before the timeout is reached.

But for these additional properties of retry policies, I think Temporal already sets pretty good defaults for most use cases. And those are what we're looking at here: The first retry happens 1 second after the initial activity failure, and the interval doubles from there until it caps off at 100 seconds between each attempt.
-->

---

## Activities: Weathering System Outages (continued)

```go
func (w *Worker) PurchaseItem(ctx workflow.Context, req PurchaseItemRequest) (*PurchaseItemResponse, error) {
    // ... generate a charge request for the item & customer

    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
    })

    // ... execute the ChargePayment activity
}
```

<!--
So the last tweak we'll make to the activity options is simply to remove the retry policy, and use the Temporal default of unlimited attempts and exponential backoff.

Just to be clear: I'm not saying you should never specify retry policy or that it is necessary to always have a long schedule to close timeout. But it is important to have a target in mind for how long your activity should be capable of retrying during a disruption, and to verify that your retry policy meets that goal.
-->

---

## Activities: Implementing Idempotency

> Idempotence is the property of certain operations in mathematics and computer science whereby they can be applied multiple times without changing the result beyond the initial application.

[wikipedia.org/wiki/Idempotence](https://en.wikipedia.org/wiki/Idempotence)

<!--
A term that gets thrown around a lot when talking about activities is idempotency. This just means that if you run an operation more than once with the same input, you should get the same result.

It turns out this is a really important property for activities to have, because they're getting retried all the time. And we really don't want to do something like charging a customer 20 times for the same purchase just because there was a temporary system outage.
-->

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

<!-- 
Unfortunately implementing and testing for idempotency is still not a solved problem. But there are some common techniques, and if you're lucky your activities are interacting with external services which themselves are designed for idempotency.

In our activities, we still might need to come up with a stable identifier that remains the same across all attempts in order to deduplicate requests to downstream services or resources that the activity creates.

Sometimes making an activity idempotent is really hard, until you split it up into multiple activities that are each invoked separately in the workflow.
-->

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
- ~~Setting `MaxAttempts: 1` in retry policy~~

<!-- I also want to note that, just in case you're thinking that it's ok to not have idempotent behavior if you set MaxAttempts to 1 in your retry policy, you should be aware that Temporal does not guarantee exactly once execution for activities. This has to do with the way server replication and failover works, so it's probably ok if you're self-hosting a single Temporal cluster but even that could change. -->

---

## Activities: Idempotency Keys

<!-- Back to the previous payment code example. Update the code to compute an idempotency key (using activityhelpers.GetIdempotencyToken) and add it to the request header (follow the example from stripe docs: https://docs.stripe.com/api/idempotent_requests). -->
```go
func (w *Worker) ChargePayment(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
    httpReq := newPaymentHTTPReq(req) // POST api.example.com/v1/payments/charge
    httpReq.Header.Set("Idempotency-Key", getIdempotencyToken(ctx))

    // ... send the HTTP request & handle errors
}

func getIdempotencyToken(ctx context.Context) string {
	info := activity.GetInfo(ctx)
	key := fmt.Sprintf("%s:%s:%s", info.WorkflowExecution.ID, info.WorkflowExecution.RunID, info.ActivityID)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
}
```

<!--
In our payment example from earlier, the activity is calling an API that supports the `Idempotency-Key` HTTP header.

So here I've add a function to generate an opaque string based on the workflow ID and activity ID, which will be the same across every activity attempt, while still being unique in case the same workflow scheduled multiple payment activities.

And then we can just pass that along to the payments API!
-->

---

## Activities: Natural Idempotency

<!-- TODO(jlegrone): rehearse the transition from the payments example to this k8s example so the use-case shift lands smoothly during the talk. -->

<!-- New code example: An activity called RunKubernetesJob that starts a k8s job and waits for it to complete (two k8s API calls). The activity should accept a struct with Name and Namespace fields, and return a struct with a Status field (completed or failed) -->
```go
func (w *Worker) RunKubernetesJob(ctx context.Context, req RunJobRequest) (*RunJobResponse, error) {
    jobs := w.client.BatchV1().Jobs(req.Namespace)
    // Create the job
    if _, err := jobs.Create(ctx, &batchv1.Job{
        Name: req.Name,
    }); err != nil {
        return nil, err
    }

    // Poll for final status
    for {
        activity.RecordHeartbeat(ctx)
        j, err := jobs.Get(ctx, req.Name)
        if err != nil { return nil, err }
        if status := getJobStatus(j); status.IsTerminal() {
            return &RunJobResponse{Status: status}, nil
        }
        time.Sleep(15 * time.Second)
    }
}
```

<!-- In the real world, we also tend to have activities that perform multiple operations and interact with systems that don't have nice primitives for idempotency. Like in this new example, where our activity is responsible for creating a Kubernetes job, and then waiting for it to complete before reporting the final job status.

The problem here is that if the activity fails or the worker is redeployed during the for loop, then on the next attempt the activity will fail at creating a job with the same name instead of skipping creation and getting the status of the existing job. No matter how many times the activity is retried, it would keep hitting that same error.
-->

---

## Activities: Natural Idempotency (continued)

<!-- Updated code example, now ignoring an already exists error for the job. -->
```diff
 func (w *Worker) RunKubernetesJob(ctx context.Context, req RunJobRequest) (*RunJobResponse, error) {
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

<!-- So the smallest fix is to add an already exists check to the job create step in the activity. I left it out here, but we'd probably also want to verify that the existing job spec matches our expected spec and replace it if not. -->

---

## Activities: Natural Idempotency (continued)
<!-- Update code example: Split into two activities, one called StartKubernetesJob and another called AwaitKubernetesJob. -->
```go
func (w *Worker) StartKubernetesJob(ctx context.Context, req StartJobRequest) error {
    jobs := w.client.BatchV1().Jobs(req.Namespace)
    // Create the job if it doesn't exist
    _, err := jobs.Create(ctx, &batchv1.Job{ /* ... */ })
    if err != nil && !apierrors.IsAlreadyExists(err) {
        return err
    }
    return nil
}

func (w *Worker) AwaitKubernetesJob(ctx context.Context, req AwaitJobRequest) (*AwaitJobResponse, error) {
    // Poll for final status
    // ...
}
```

<!-- Splitting responsibilities across two separate activities is probably an even cleaner approach though, because now we can set independent timeouts and retry policies for each step of creating the job, and then waiting for it to complete. We also automatically get more visibility into what's going on in the workflow history.

Note that I still kept the already exists check in the new `StartKubernetesJob` activity though. Even though it's not likely, a network partition or a poorly timed worker crash could still mean that the activity is retried even after the Create API call succeeds. But reducing the scope of what the activity does still makes reasoning about idempotency a bit simpler.
-->

---

## Activities: Handling Worker Disruptions

Choosing between StartToClose and Heartbeat timeouts

```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunJobRequest) (*RunJobResponse, error) {
    // Execute the StartKubernetesJob activity
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            StartToCloseTimeout:    time.Hour,
            ScheduleToCloseTimeout: time.Hour,
        }),
        w.AwaitKubernetesJob,
        AwaitJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

<!--
In the workflow that invokes our activities to create and wait for completion of the Kubernetes job, we need to choose some timeout values. Here we went with a 1 hour StartToClose and ScheduleToClose timeout because we wanted to allow the Kubernetes job to run for up to an hour, and the activity itself shouldn't be timed out as long as it's still polling the job status in that for loop.

The only problem here is that if the worker crashes or terminates unexpectedly, then it will never report a final result or error for the activity. And since the ScheduleToClose timeout is the same duration as StartToClose timeout, Temporal won't retry the activity on our behalf either.

We could decrease the StartToClose timeout and allow the activity to time out and be retried. But that forces a tradeoff between how quickly Temporal retries after a worker failure and how often we are doing unnecessary retries (which in some cases, could mean repeating expensive work).
-->

---

## Activities: Handling Worker Disruptions (continued)

<!-- Updated code example: Replace the start to close timeout with a 30s heartbeat timeout.

Speaker notes:
- Replacing a start to close timeout with heartbeat timeout avoids the tradeoff between retrying quickly when the worker fails, and allowing your longest-running tasks to complete. Now the activity can run as long as it needs to, up to the schedule to close timeout, but is retried quickly if the worker becomes unresponsive.
 -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunJobRequest) (*RunJobResponse, error) {
    // Execute the StartKubernetesJob activity
    // ...

    // Wait for the job to complete
    return workflowhelpers.AwaitActivity(
        workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
            // This activity may run for a long time, so use Heartbeat instead of StartToClose timeout.
            HeartbeatTimeout:       30 * time.Second,
            ScheduleToCloseTimeout: time.Hour,
        }),
        w.AwaitKubernetesJob,
        AwaitJobRequest{Name: req.Name, Namespace: req.Namespace},
    )
}
```

<!--
The more elegant solution is to swap out the StartToClose timeout for a Heartbeat timeout.

This allows our activity to run for as long as it needs to, up to the ScheduleToClose timeout, without being stopped and retried, as long as it keeps reporting back to the Temporal server via heartbeat messages. If the worker fails to send heartbeats, then Temporal can also retry the activity.

When choosing a value for Heartbeat or StartToClose timeout, the question to answer is how long you can wait for the activity's retry policy to kick in if the worker fails. My default is to use 30s, because that's about how long I reasonably want to be stuck staring at a workflow in the Temporal UI waiting to see if the activity is still in progress or the worker has become unresponsive.
-->

---

## Activities: A Grand Unified Theory

1. Activities should be idempotent.
2. Always set a **ScheduleToClose** timeout. Base the value on how long the activity should continue retrying during a worst case outage.
3. Always set either **Heartbeat** or **StartToClose** timeout. Use **StartToClose** timeout only when the activity is guaranteed to not run past that duration and it is acceptable to wait the whole duration before a retry. 
4. Activities that perform cleanup on cancelation MUST send heartbeats.
5. Prefer unlimited attempts with `ScheduleToClose` as the bound.
6. Respect error conventions from downstream services. Translate these into `TemporalApplicationError` to skip retry or adjust backoff behavior.

** Consider implementing and/or enforcing these policies in an interceptor.

<!--
There are plenty more mistakes to make around activities that we can't cover here, but the good news is that I think almost all of them can be avoided by following this smallish set of guidelines.

Granted not all of these are easy to follow. And you'll probably be hard pressed to get a coding assistant to always come up with perfectly congruous timeouts and retry policies or always translate status codes from an HTTP response into Temporal application errors.

That's why I've also been working on a specification for a Temporal worker interceptor that enforces good timeout and retry policies, and gRPC and HTTP middleware for translating common errors codes into Temporal errors with appropriate retry behavior. You can find Go and Python reference implementations of this spec in the 100-temporal-mistakes repo.
-->

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
