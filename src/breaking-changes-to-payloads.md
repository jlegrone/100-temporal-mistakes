# Breaking Changes to Payloads

> [!TIP]
> Workflow and activity inputs/outputs, side effects, [signal](terms/signals.md) payloads, and [query](terms/queries.md) parameters are serialized and stored in [history](terms/event-history.md) -- treat their types like a wire protocol. Renaming fields, changing types, or removing fields can break deserialization for in-flight workflows after a worker deployment.

Every [payload](terms/payload.md) type that flows through Temporal is serialized via a [data converter](terms/data-converter.md) and persisted in the workflow history. When a running workflow replays, those payloads are deserialized using the *current* version of your code. A breaking change -- renaming a field (e.g., `customer_id` to `customerId`), changing a field's type, removing a field, or reordering positional arguments -- means the new code can no longer correctly deserialize values written by a previous version.

Treat payload types with the same discipline you'd apply to a database schema or a public API contract. Add optional fields with sensible defaults; don't remove or rename existing ones. If a field must change type, add a new field alongside the old one and handle both during deserialization. Use schema-friendly serialization like Protobuf, which has explicit rules for backwards-compatible evolution. When a payload change is necessary, carefully validate backwards compatibility via replay testing.

See also: [Lossy payload serialization](lossy-payload-serialization.md), [Using more than one input/response payload](using_more_than_one_input_response_payload/).
