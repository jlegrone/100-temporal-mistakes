# Lossy Payload Serialization

<!-- TODO: find documentation on JSON serialization and round-trip testing. Then delete and replace the -->
> [!TIP]
> TODO

Temporal serializes all workflow and activity inputs and outputs using a [data converter](terms/data-converter.md). By default, most SDKs use JSON, which is a lossy format for many common types, like floats.

To avoid this, understand what your SDK's default data converter does with each type you use, and test round-trips explicitly. Prefer simple, unambiguous types in your workflow and activity signatures: strings for dates (ISO 8601), strings for enums, and no language-specific types without clean JSON representations. If your application needs richer types, implement a custom [data converter](terms/data-converter.md) that preserves type information -- Protocol Buffers are a good option here. Finally, run replay tests using captured workflow histories to catch serialization mismatches before they reach production.

See also: [Breaking changes to payloads](breaking-changes-to-payloads.md), [Not using workflow replay for debugging](not_using_workflow_replay_for_debugging/).

<!-- TODO: The most robust way to ensure safe serialization is by using a format like protobuf where types are guaranteed to round-trip. This also can help with cross language worker/client compatibility. Link to the temporal community Proto plug-in, but mention that a Proto plug-in is not required just to use Protobuf messages in workflow and activity payloads. -->
<!-- TODO: add a code example for round-trip payload serde testing in go. -->
