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

<!-- Updated code example that also increases the next retry backoff time when external service returns a resource overloaded error (HTTP 429) using the activityhelpers.GetNextRetryDelay function and multiplying its return value by 1.5. -->
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

## Activities: Implementing Idempotency

Three techniques to achieve idempotency:
- Passing idempotency key to external APIs
    - Derive a key from the Workflow ID and activity ID. Pass this to downstream systems (like Stripe) to ignore duplicate requests.
- Applying database constraints
    - Use `INSERT ... ON CONFLICT` or conditional writes to ensure records aren't created twice.
- Using naturally idempotent operations
    - Design side effects as state settings (Set to X) rather than increments (+1), or use upserts with fixed IDs.
    - May help to decompose into multiple activities.

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

## Activities: Idempotency Keys

<!-- Back to the previous payment code example. Update the code to compute an idempotency key (using activityhelpers.GetIdempotencyToken) and add it to the request header (follow the example from stripe docs: https://docs.stripe.com/api/idempotent_requests). -->
```go
func getIdempotencyToken(ctx context.Context) string {
	info := activity.GetInfo(ctx)
	key := fmt.Sprintf("%s:%s:%d", info.WorkflowExecution.ID, info.WorkflowExecution.RunID, info.ActivityID)
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

## Activities: Handling Worker Disruptions

Choosing between StartToClose and Heartbeat timeouts

<!-- New workflow code example, this time invoking our (longer running) AwaitKubernetesJob activity. Set a 30s start to close timeout and a 1h schedule to close timeout. -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    // Start the job
    
    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
    })
    return workflowhelpers.AwaitActivity(ctx, w.AwaitKubernetesJob, req)
}
```

<!-- Speaker note: Our previous activity example was expected to always complete in under 30s. But that's not the case for all activities. A short start to close timeout made sense for the previous use case, but what about for an activity that could run for much longer? Setting too short a value could mean that some requests never complete, no matter how many retry attempts are made. -->

<!-- Updated code example: Change the start to close timeout to 5m.

Speaker note:
So we can try increasing the start to close timeout, but now this also means that if the worker crashes or becomes unresponsive, we'd have to wait much longer before Temporal retries the activity.
-->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout:    5 * time.Minute,
        ScheduleToCloseTimeout: time.Hour,
    })
    var resp AwaitKubernetesJobResponse
    if err := workflow.ExecuteActivity(ctx, w.AwaitKubernetesJob, req).Get(ctx, &resp); err != nil {
        return nil, err
    }
    // ... return the result ...
}
```

<!-- Updated code example: Replace the start to close timeout with a 30s heartbeat timeout.

Speaker notes:
- Replacing a start to close timeout with heartbeat timeout avoids the tradeoff between retrying quickly when the worker fails, and allowing your longest-running tasks to complete. Now the activity can run as long as it needs to, up to the schedule to close timeout, but is retried quickly if the worker becomes unresponsive.
 -->
```go
func (w *Worker) RunKubernetesJob(ctx workflow.Context, req RunKubernetesJobRequest) (*RunKubernetesJobResponse, error) {
    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        HeartbeatTimeout:       30 * time.Second,
        ScheduleToCloseTimeout: time.Hour,
    })
    var resp AwaitKubernetesJobResponse
    if err := workflow.ExecuteActivity(ctx, w.AwaitKubernetesJob, req).Get(ctx, &resp); err != nil {
        return nil, err
    }
    // ... return the result ...
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

