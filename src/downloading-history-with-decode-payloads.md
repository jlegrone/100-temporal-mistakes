# Downloading History with DecodePayloads Enabled

> [!TIP]
> * Downloading [workflow history](terms/event-history.md) from the Temporal UI with "Decode Payloads" enabled produces a modified history that cannot be used for replay testing.
> * Always download the raw (encoded) history when you need it for [replay](terms/replay.md) tests or workflow reset operations.
> * The decoded version is only useful for human readability and debugging, not as a replay input.

## What?

The Temporal Web UI offers an option to download workflow history as JSON. When you toggle the "Decode Payloads" option, the UI uses the configured [data converter](terms/data-converter.md) (or codec server) to decode all [payloads](terms/payload.md) in the history before saving the file. The resulting JSON contains human-readable data instead of the raw base64-encoded payloads.

While this is convenient for reading the history, the decoded file has a different structure than what the SDK expects during [replay](terms/replay.md). Using a decoded history file for replay testing will either fail outright or produce incorrect results.

## Why?

Temporal persists all workflow and activity inputs, outputs, and other payloads in their serialized (encoded) form. The [data converter](terms/data-converter.md) encodes data before it goes to the server and decodes it when it comes back. When the UI decodes payloads for download, it bakes the decoding step into the file, altering the payload structure.

During replay, the SDK reads history events and feeds them through its own data converter to deserialize payloads. If the payloads are already decoded, the data converter either fails to parse them or double-decodes them, leading to:
- Replay test failures with deserialization errors
- Silently incorrect data if the double-decoding happens to produce valid but wrong output
- Workflow reset failures when the modified history is used as input

## Solution

**For replay testing and workflow reset:** always download history with "Decode Payloads" **disabled**. This gives you the raw history exactly as the Temporal server stores it, which is what the SDK expects.

**For debugging and human inspection:** use the decoded version freely. It helps you understand what data flowed through the workflow, but treat it as a read-only artifact.

**In CI/CD pipelines:** when programmatically fetching history for replay tests (e.g., via `tctl` or the SDK client), the default behavior returns raw payloads. Avoid passing any decode flags when fetching history for automated replay testing.

```bash
# Correct: raw history for replay testing
tctl workflow show --workflow_id my-workflow --run_id abc123 --output_filename history.json

# Only for human reading, NOT for replay
tctl workflow show --workflow_id my-workflow --run_id abc123 --output_filename history_readable.json --decode_payloads
```
