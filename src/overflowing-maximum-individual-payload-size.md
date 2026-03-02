# Overflowing Maximum Individual Payload Size

> [!TIP]
> - Temporal serializes every workflow and activity input and output, then sends it over gRPC to the server.
> - Individual [payloads](terms/payload.md) cannot exceed 4MB by default.

## What?

Every workflow and activity input and output is serialized by the [data converter](terms/data-converter.md), sent over the network, and stored in the [Temporal server backend](terms/temporal-server-backend.md). This means hard size limits apply to each payload.

By default, a serialized input or output cannot exceed 2MB. Larger payloads cause the server to reject the request with a `ResourceExhausted` error. The workflow stops making progress and your [worker](terms/worker.md) logs errors.

## Why?

The Temporal server exposes a [gRPC](https://grpc.io) API built on [unary RPCs](https://grpc.io/docs/what-is-grpc/core-concepts/#unary-rpc) (request-reply). gRPC enforces a [default message size limit of 4MB](https://github.com/grpc/grpc-java/issues/1676#issuecomment-229809402), which constrains workflow and activity payloads since they are embedded in gRPC requests.

Temporal API requests also carry metadata alongside your payloads, which counts toward the 4MB limit.

## Solution

1. Trim inputs and outputs to the minimum. Return only what the caller needs.
2. Store large data in an external system (a database, blob storage, or local disk via [sessions](terms/sessions.md)) and pass references (IDs, URLs) as your workflow and activity inputs.
3. Use the [large payload codec](terms/large-payload-codec.md) if only a small fraction of your traffic exceeds gRPC limits.
4. For data-intensive workloads, consider purpose-built systems instead of Temporal.
