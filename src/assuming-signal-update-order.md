# Assuming Signal and Update Delivery Order

> [!TIP]
> * [Signals](terms/signals.md) and [updates](terms/updates.md) are delivered asynchronously and may arrive in a different order than they were sent.
> * Even when sent sequentially from a single client, network conditions and server processing can reorder them.
> * Design workflows to tolerate any delivery order using sequence numbers, timestamps, or state machines.

## What?

Developers often assume that if they send signal A before signal B from the same client, the workflow will process A before B. This assumption is incorrect. [Signals](terms/signals.md) and [updates](terms/updates.md) are asynchronous by nature, and Temporal does not guarantee delivery order across multiple signal or update calls.

The same applies to mixed operations: if you send a signal and then an update (or vice versa), there's no guarantee about which one the workflow processes first.

## Why?

Several factors can cause reordering:

- **Network conditions.** Different requests may take different paths through load balancers and network infrastructure, arriving at the server in a different order than sent.
- **Server processing.** The Temporal server processes requests concurrently. Two requests arriving close together may be processed in any order.
- **Retries.** If a signal delivery fails and the client retries, the retry may arrive after a later signal.

When your workflow logic depends on a specific ordering (e.g., "initialize" must come before "process", or events must be applied in sequence), out-of-order delivery leads to incorrect state, dropped data, or workflow failures.

## How?

Design your workflow to handle messages in any order:

1. **Sequence numbers.** Include a monotonically increasing sequence number in each signal [payload](terms/payload.md). Buffer out-of-order messages and process them in sequence order.

    ```go
    // In the signal handler, buffer and process in order
    if msg.SeqNum == expectedSeqNum {
        process(msg)
        expectedSeqNum++
        // Drain any buffered messages that are now in order
    } else {
        buffer[msg.SeqNum] = msg
    }
    ```

2. **Timestamps.** If exact ordering isn't critical but recency matters, include timestamps and use last-write-wins or similar conflict resolution strategies.

3. **State machines.** Model your workflow state as a state machine that validates transitions. Reject or buffer messages that aren't valid for the current state instead of assuming they'll arrive in the right order.

4. **Single entry point.** If ordering truly matters and you control the sender, consider sending a single signal with a batch of ordered operations rather than multiple individual signals.
