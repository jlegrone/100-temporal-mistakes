# Overflowing Maximum Individual Payload Size

> [!TIP]
> Individual [payloads](terms/payload.md) cannot exceed 4MB by default. The Temporal server exposes a [gRPC](https://grpc.io) API with a [default message size limit of 4MB](https://github.com/grpc/grpc-java/issues/1676#issuecomment-229809402), and API request metadata also counts toward that limit.

Every workflow and activity input and output is serialized by the [data converter](terms/data-converter.md), sent over gRPC, and stored in the [Temporal server backend](terms/temporal-server-backend.md). Payloads that exceed the gRPC size limit are rejected with a `ResourceExhausted` error, causing the workflow to stop making progress while your [worker](terms/worker.md) logs errors.

To avoid this, trim inputs and outputs to the minimum the caller needs. Store large data in an external system (a database, blob storage, or local disk via [sessions](terms/sessions.md)) and pass references (IDs, URLs) as your workflow and activity inputs. Use the [large payload codec](terms/large-payload-codec.md) if only a small fraction of your traffic exceeds gRPC limits. For data-intensive workloads, consider whether Temporal is the right tool or if a purpose-built data pipeline is more appropriate.

See also: [Passing too much information from activities](passing-too-much-information-from-activities.md), [Overflowing workflow history bytes](overflowing-workflow-history-bytes.md).
