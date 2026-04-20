# Assuming Signal and Update Delivery Order

> [!TIP]
> [Signals](terms/signals.md) and [updates](terms/updates.md) may arrive in a different order than sent, even from a single client. Design workflows to tolerate any delivery order.

Developers often assume that sending signal A before signal B guarantees the workflow processes A first. This is incorrect. Network conditions, server-side concurrency, and client retries can all cause reordering. When workflow logic depends on specific ordering ("initialize" before "process"), out-of-order delivery leads to incorrect state or failures.

Design for any order: include sequence numbers in signal [payloads](terms/payload.md) and buffer out-of-order messages, use timestamps with last-write-wins resolution, model your workflow as a state machine that validates transitions, or send a single signal with a batch of ordered operations when ordering truly matters.

```go
if msg.SeqNum == expectedSeqNum {
    process(msg)
    expectedSeqNum++
    // Drain any buffered messages that are now in order
} else {
    buffer[msg.SeqNum] = msg
}
```
