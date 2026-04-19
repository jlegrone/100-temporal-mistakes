# Not Sending Heartbeats From Activities You Want To Handle Cancellation

> [!TIP]
> * Activities must send heartbeats to the server for them to be cancelled in a timely when their calling workflow is.

## What?

In Temporal, to prevent a workflow from making progress, you have two choices: termination and cancellation. While termination is a hammer that immediately stops the workflow, cancellation is a nicer alternative which allows workflows to perform cleanup operations before completing.

While termination comes for free and works as expected out of the box, cancellation is more involved process which require some setup to ensure a workflow and its activities can respond to cancellation signals properly.

One of that surprising requirement is that for an activity to be cancelled, it MUST send regular heartbeats to the server. If not, the workflow will wait for the activity to complete (successfully or with an error like a timeout) before being cancelled which is often not the desired outcome when you cancel a workflow.

## Why?

The reason heartbeating from activities is mandatory is Temporal workflows and activities execute independently from the server on Temporal workers, an entirely different process. Cancellation signals are sent to a server which has to relay that information somehow to workers. Servers can't directly send information to workers due to how Temporal is designed. Servers can only communicate with workers through their task queue with the exception of heartbeats.

Heartbeat API is a Temporal server endpoint that can be called from within actiity code. That API is a way for activities to send regular checkpoints to the server so they can:
- Inform the server they are still running
- Provide heartbeat details so they can resume from a point of interruption if they are aborted mid-way.

As that API call is synchronous, the server has the hability to send back a response to the activity and that response can carry over the information that the workflow has been cancelled. This is the only way for server to transmit that information as other communication with workers only occurs asynchronously over workers task queue.

## How?

Activities must regularly call the [activity.RecordHeartbeat](https://pkg.go.dev/go.temporal.io/sdk@v1.30.0/activity#RecordHeartbeat) (or equivalent in other SDKs) so they can receive the cancelation signal.

Hearbeats must be send more frequently than the [activity heartbeat timeout](https://temporal.io/blog/activity-timeouts#heartbeat-timeout)) or the activity will be marked as failed if they don't.

The time between each heartbeat will directly impact the responsivness of your cancellation. If you hearbeat every hour in a long running activity, that activity may be cancelled only after that amount of time has elapsed making your workflow appear as it was not responding to cancellation. You have to send heartbeat more frequently if you need cancellation to propagate more quickly, your activity heartbeat timeout capping the maximum amount of time for the signal to propagate.

If you don't care about cancelling ongoing activities and are fine waiting for them to complete when you cancel a workflow, heartbeating is not an absolute requirement. This might be the case for short running activities. In general though, long running activities should always send heartbeats regularly so they can be resumed promptly when workers are deployed in the middle of their execution.
