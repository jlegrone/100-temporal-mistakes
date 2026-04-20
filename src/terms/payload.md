# Payload

A payload is the serialized form of data passed as inputs and outputs of workflows, activities, signals, updates, and queries. Payloads are created by the data converter, transmitted over gRPC to the Temporal server, and persisted in the workflow's event history.

Because payloads are persisted and transmitted, their size directly impacts history size, replay performance, gRPC message limits (default 4MB), and storage costs. Best practice is to keep payloads small -- pass IDs and references rather than full data objects.

## Related

- [Overflowing Maximum Individual Payload Size](../overflowing-maximum-individual-payload-size.md)
- [Passing Too Much Information from Activities](../passing_too_much_information_from_activities/)
- [Lossy Payload Serialization](../lossy-payload-serialization.md)
- [Breaking Changes to Payloads](../breaking-changes-to-payloads.md)
- [Data Converter](data-converter.md)
- [Large Payload Codec](large-payload-codec.md)
- [Event History](event-history.md)
