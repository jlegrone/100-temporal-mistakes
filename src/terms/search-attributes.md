# Search Attributes

Search attributes are custom key-value pairs attached to a workflow execution that can be used for filtering and querying through the visibility APIs. They are indexed separately from the event history and can be updated during workflow execution using `UpsertSearchAttributes`.

Search attributes are the recommended way to expose workflow state for querying, as an alternative to using queries (which require replaying the workflow) or the ListWorkflow API for business logic. Common use cases include tagging workflows with customer IDs, order statuses, or other business-relevant metadata.

## Related

- [Depending on List Workflow API](../depending-on-list-workflow-api.md)
- [Querying Closed Workflows](../querying-closed-workflows.md)
- [Storing Sensitive Data in Workflow History](../storing-sensitive-data-in-workflow-history.md)
- [Visibility](visibility.md)
- [Queries](queries.md)
- [Event History](event-history.md)
