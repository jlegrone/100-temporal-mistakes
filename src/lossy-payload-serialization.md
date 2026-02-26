# Lossy Payload Serialization

> [!TIP]
> * Serialization round-trips that lose type information (dates becoming strings, enums becoming ints, custom types flattened) can cause subtle bugs during [replay](terms/replay.md).
> * The default JSON serializer in most SDKs is convenient but not always lossless -- know its limitations.
> * Use a [data converter](terms/data-converter.md) that preserves type fidelity, or design your [payload](terms/payload.md) types to survive round-trips cleanly.

## What?

Temporal serializes all workflow and activity inputs and outputs using a [data converter](terms/data-converter.md). By default, most SDKs use JSON. The problem is that JSON is a lossy format for many common types:

- `time.Time` in Go serializes to a string -- the original type information is gone.
- Enums may serialize as integers with no indication of the enum type.
- Language-specific types like Python's `datetime`, `Decimal`, or `set` lose their identity through JSON.
- Nested structs may deserialize into generic maps or dictionaries instead of typed objects.

This works fine during the initial execution because the code naturally handles the types it expects. But during [replay](terms/replay.md), the SDK deserializes values from [history](terms/event-history.md), and the reconstituted types may not match what the code originally produced.

## Why?

Replay depends on the ability to faithfully reconstruct the state of a workflow execution from its history. When deserialized values don't match the original types, several things can go wrong:

1. **Comparison failures**: A value that was a `time.Time` is now a `string`. Comparisons or arithmetic on it may behave differently or fail.
2. **Type assertion errors**: Code that type-asserts or pattern-matches on the deserialized value may fail during replay even though it worked during the original execution.
3. **Silent behavior changes**: The worst case -- the code doesn't crash but takes a different branch because the deserialized value is subtly different (e.g., float precision loss, timezone handling differences).

These bugs are particularly hard to diagnose because they only appear during replay, not during normal execution.

## Solution

1. **Know your serializer's limitations**: Understand what your SDK's default data converter does with each type you pass. Test round-trips explicitly if you're unsure.

2. **Design payload types for safe serialization**: Prefer simple, unambiguous types in your workflow and activity signatures. Use strings for dates (ISO 8601), strings for enums, and avoid language-specific types that don't have clean JSON representations.

3. **Use a custom data converter**: If your application needs richer types, implement a custom [data converter](terms/data-converter.md) that preserves type information. For example, Protocol Buffers (protobuf) provide a schema-based format that avoids most of these issues.

4. **Test replay explicitly**: Run replay tests (using workflow history from actual executions or captured fixtures) to catch serialization mismatches before they hit production. This pairs well with [using workflow replay for debugging](not-using-workflow-replay-for-debugging.md).
