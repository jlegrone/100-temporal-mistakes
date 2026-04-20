# Downloading History with DecodePayloads Enabled

> [!TIP]
> Downloading [workflow history](terms/event-history.md) from the Temporal UI with "Decode Payloads" enabled produces a modified history that cannot be used for [replay](terms/replay.md) testing or workflow reset. Always download the raw (encoded) history for these purposes.

The Temporal Web UI offers an option to download workflow history as JSON. When you toggle "Decode Payloads," the UI uses the configured [data converter](terms/data-converter.md) (or codec server) to decode all [payloads](terms/payload.md) before saving the file. The resulting JSON contains human-readable data instead of raw base64-encoded payloads. While convenient for reading, the decoded file has a different structure than what the SDK expects during replay. Using it for replay testing will either fail outright with deserialization errors or produce silently incorrect results from double-decoding.

For replay testing and workflow reset, always download history with "Decode Payloads" disabled. This gives you the raw history exactly as the Temporal server stores it. For debugging and human inspection, the decoded version is fine -- just treat it as a read-only artifact. In CI/CD pipelines that programmatically fetch history for replay tests (via `temporal` CLI or the SDK client), the default behavior returns raw payloads; avoid passing any decode flags.

```bash
# Correct: raw history for replay testing
temporal workflow show --workflow-id my-workflow --run-id abc123 --output json > history.json

# Only for human reading, NOT for replay
temporal workflow show --workflow-id my-workflow --run-id abc123 --output json --codec-endpoint http://localhost:8081 > history_readable.json
```

See also: [Not Using Workflow Replay for Debugging](not_using_workflow_replay_for_debugging/), [Not Validating Replay Safety Before Deployments](not_validating_replay_safety_before_deployments/README.md).
