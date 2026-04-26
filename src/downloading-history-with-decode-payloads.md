# Downloading History with DecodePayloads Enabled

<!-- TODO: Add screenshot of correct settings from the temporal UI modal -->

> [!TIP]
> Downloading [workflow history](terms/event-history.md) from the Temporal UI with "Decode Payloads" enabled produces a modified history that cannot be used for [replay](terms/replay.md) testing or workflow reset. Always download the raw (encoded) history for these purposes.

The Temporal Web UI offers an option to download workflow history as JSON. When you toggle "Decode Payloads," the UI uses the configured [data converter](terms/data-converter.md) (or codec server) to decode all [payloads](terms/payload.md) before saving the file. The resulting JSON contains human-readable data instead of raw payloads. Using it for replay testing will often fail with deserialization errors that do not occur in production.

For replay testing and workflow reset, always download history with "Decode Payloads" disabled. This gives you the raw history exactly as the Temporal server stores it.

<!-- TODO: ONLY show the command to download with the correct for replaying format -->

```bash
# Correct: raw history for replay testing
temporal workflow show --workflow-id my-workflow --run-id abc123 --output json > history.json

# Only for human reading, NOT for replay
temporal workflow show --workflow-id my-workflow --run-id abc123 --output json --codec-endpoint http://localhost:8081 > history_readable.json
```

See also: [Not Using Workflow Replay for Debugging](not_using_workflow_replay_for_debugging/), [Not Validating Replay Safety Before Deployments](not_validating_replay_safety_before_deployments/README.md).
