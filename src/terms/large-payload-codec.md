# Large Payload Codec

The Large Payload Codec is a codec implementation that transparently offloads oversized payloads to external blob storage (e.g., S3) and replaces them with a reference in the workflow history. This allows workflows to work with data that would otherwise exceed gRPC message size limits. The codec operates as a layer within the Data Converter pipeline, so it is transparent to workflow and activity code.

## Related

- [Overflowing maximum individual payload size](../overflowing-maximum-individual-payload-size.md)
- [Overflowing workflow history bytes](../overflowing-workflow-history-bytes.md)
- [Data Converter](data-converter.md)
- [Payload](payload.md)
- [Temporal Server Backend](temporal-server-backend.md)
