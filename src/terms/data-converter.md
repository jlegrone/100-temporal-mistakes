# Data Converter

A Data Converter is the component responsible for serializing and deserializing workflow and activity inputs and outputs (payloads). The SDK provides a default JSON-based data converter, but custom converters can be implemented for different serialization formats, encryption, or compression. A custom data converter with encryption is the standard approach for protecting sensitive data in workflow history.

## Related

- [Lossy payload serialization](../lossy-payload-serialization.md)
- [Breaking changes to payloads](../breaking-changes-to-payloads.md)
- [Overflowing maximum individual payload size](../overflowing-maximum-individual-payload-size.md)
- [Storing sensitive data in workflow history](../storing-sensitive-data-in-workflow-history.md)
- [Downloading history with decode payloads](../downloading-history-with-decode-payloads.md)
- [Payload](payload.md)
- [Large Payload Codec](large-payload-codec.md)
