# Lossy Payload Serialization

> [!TIP]
> The default JSON serializer in most SDKs loses type information on round-trips (dates become strings, enums become ints, custom types are flattened), which can cause subtle bugs during [replay](terms/replay.md).

Temporal serializes all workflow and activity inputs and outputs using a [data converter](terms/data-converter.md). By default, most SDKs use JSON, which is a lossy format for many common types: `time.Time` in Go serializes to a string, enums may serialize as bare integers, Python's `datetime`, `Decimal`, or `set` lose their identity, and nested structs may deserialize into generic maps. This works fine during initial execution, but during [replay](terms/replay.md) the SDK deserializes values from [history](terms/event-history.md), and the reconstituted types may not match what the code originally produced. The result can be comparison failures, type assertion errors, or -- worst of all -- silent behavior changes where the code takes a different branch because a deserialized value is subtly different.

To avoid this, understand what your SDK's default data converter does with each type you use, and test round-trips explicitly. Prefer simple, unambiguous types in your workflow and activity signatures: strings for dates (ISO 8601), strings for enums, and no language-specific types without clean JSON representations. If your application needs richer types, implement a custom [data converter](terms/data-converter.md) that preserves type information -- Protocol Buffers are a good option here. Finally, run replay tests using captured workflow histories to catch serialization mismatches before they reach production.

See also: [Breaking changes to payloads](breaking-changes-to-payloads.md), [Not using workflow replay for debugging](not_using_workflow_replay_for_debugging/).
