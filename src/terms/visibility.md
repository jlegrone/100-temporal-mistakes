# Visibility

Visibility is Temporal's system for querying and listing workflow executions based on their metadata. It enables searching workflows by status, type, time range, and custom search attributes. Visibility data is stored separately from workflow event histories and may have different consistency guarantees depending on the backend.

Standard visibility (backed by the same database as persistence) provides basic listing. Advanced visibility (backed by Elasticsearch or similar) supports complex queries, custom search attributes, and faster listing. Visibility APIs are designed for operational and debugging use cases, not for application logic, as they have eventual consistency.

## Related

- [Depending on List Workflow API](../depending-on-list-workflow-api.md)
- [Querying Closed Workflows](../querying-closed-workflows.md)
- [Underutilizing Namespaces](../underutilizing-namespaces.md)
- [Search Attributes](search-attributes.md)
- [Namespace](namespace.md)
- [Temporal Server Backend](temporal-server-backend.md)
