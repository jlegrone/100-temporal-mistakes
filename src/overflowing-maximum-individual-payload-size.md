# Overflowing Maximum Individual Payload Size

> [!TIP]
> - Workflows and activities inputs and outputs are constantly serialized and sent over the network to a Temporal gRPC server.
> - Workflows and activities inputs and outputs have hard limits in terms of size and can’t exceed 4MB by default.

## What?
It is important to remember that every input and output of workflows and activities are being serialized (using a [data converter](<terms/data-converter.md>), send over the network and persisted in your [temporal server backend](<terms/temporal-server-backend.md>). This means that limits in terms of payload size exist.

By default, serialized inputs/outputs can’t go larger than 2MB. If they grow larger, the request is rejected by the server with a `ResourceExhausted` error and your workflow stop making progress $^\text{[ref needed]}$ and your worker will log errors.

## Why?

Temporal servers under the hood expose a [gRPC](https://grpc.io) API that mostly consist of [unary RPCs](https://grpc.io/docs/what-is-grpc/core-concepts/#unary-rpc) and so request-reply based. As such hard limits in terms of individual request payload size exists. By default [gRPC servers set the limit at 4MB](https://github.com/grpc/grpc-java/issues/1676#issuecomment-229809402) which result in limiting the size of activities and workflows inputs as those gets embedded into a gRPC RPC call in the end.

Note that Temporal API requests have metadata in addition to the workflows and activities payloads which also counts towards the 4MB limit.

## Solution

To workaround the issue, it is recommended to:
1. Trim down your inputs and outputs to the minimum first
2. Store the payloads in an external system (a database, blob storage, disk if you use [sessions](terms/sessions.md) , …) and pass references (IDs, links, …) to as inputs to your payload.
3. Look at the [large payload codec](<terms/large-payload-codec.md>) (if you only need to deal with a small fraction of your traffic exceeding gRPC limits).
4. Consider alternatives solutions to Temporal for data intensives applications.
