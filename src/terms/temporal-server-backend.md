# Temporal Server Backend

The Temporal Server backend refers to the persistence layer that stores workflow event histories, visibility data, and cluster metadata. Supported backends include Cassandra, MySQL, and PostgreSQL. The backend's performance and availability directly impact Temporal's throughput and reliability. Proper configuration of persistence rate limits and monitoring of database health are essential for production deployments.

## Related

- [Overflowing maximum individual payload size](../overflowing-maximum-individual-payload-size.md)
- [Storing sensitive data in workflow history](../storing_sensitive_data_in_workflow_history/)
- [Dynamic Config](dynamic-config.md)
- [Event History](event-history.md)
- [Namespace](namespace.md)
- [Visibility](visibility.md)
