# Breaking Changes to Payloads

> [!TIP]
> * Workflow inputs, activity inputs/outputs, [signal](terms/signals.md) payloads, and [query](terms/queries.md) parameters are serialized and stored in [history](terms/event-history.md) -- treat their types like a wire protocol.
> * Renaming fields, changing types, or removing fields can break deserialization for in-flight workflows during [replay](terms/replay.md).
> * Use backwards-compatible changes only: add optional fields, don't remove or rename existing ones.

## What?

Every [payload](terms/payload.md) type that flows through Temporal -- workflow inputs, activity inputs and outputs, signal payloads, [update](terms/updates.md) parameters -- gets serialized via a [data converter](terms/data-converter.md) and persisted in the workflow history. When a running workflow is replayed, those historical payloads are deserialized using the *current* version of your code.

A breaking change to a payload type means that the current code can no longer correctly deserialize values that were written by a previous version. Common examples:

- Renaming a field (e.g., `customer_id` to `customerId`) -- the old value can't be found under the new name.
- Changing a field's type (e.g., `string` to `int`, or a flat field to a nested object).
- Removing a field that old workflows may have persisted.
- Reordering fields in positional serialization formats.

## Why?

Unlike a typical web API where you control both the client and server and can coordinate deployments, Temporal workflows can run for days, weeks, or months. At any point during that lifespan, a [worker](terms/worker.md) restart triggers [replay](terms/replay.md) which deserializes historical payloads using the latest code.

If the payload types have changed incompatibly, replay fails with deserialization errors. The workflow becomes stuck -- it can't make progress because it can't read its own history. Fixing this typically requires either:

- Reverting the code change (if possible).
- Using [versioning](terms/versioning.md) to maintain two code paths.
- Manually resetting or [terminating](terms/terminate.md) affected workflows.

None of these are pleasant options, especially at scale.

## How?

Treat payload types with the same discipline you'd apply to a database schema or a public API contract:

1. **Add fields, don't remove them.** New optional fields with sensible defaults are always safe. Old workflows won't have the field in their history, so your code should handle its absence.

2. **Don't rename fields.** If you must rename, add the new field alongside the old one and handle both during deserialization. Remove the old field only after all in-flight workflows that used it have completed.

3. **Don't change field types.** If a field needs to go from `string` to `int`, add a new field with the new type instead.

4. **Use schema-friendly serialization.** Protobuf and similar formats have explicit rules for backwards-compatible evolution (field numbers, optional fields). JSON with named fields is more forgiving than positional formats but still requires care.

5. **Version your workflow code.** When a payload change is truly necessary, use [versioning](terms/versioning.md) to branch your workflow logic so that old histories are processed with old types and new workflows use the new types.

6. **Test with real history.** Download history from running workflows and replay it against your new code before deploying. This catches deserialization issues before they affect production.
