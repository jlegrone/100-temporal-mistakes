# Parent Close Policy

ParentClosePolicy determines what happens to a child workflow when its parent workflow execution completes, fails, times out, is cancelled, or is terminated. There are three options:

- **TERMINATE** (default): The child workflow is immediately terminated. No cleanup code runs.
- **REQUEST_CANCEL**: A cancellation request is sent to the child workflow, giving it a chance to perform cleanup.
- **ABANDON**: The child workflow continues running independently, unaffected by the parent's completion.

The default TERMINATE policy catches many teams off guard when they expect child workflows to complete their work after the parent finishes.

## Related

- [Not Using Parent Close Policy](../not-using-parent-close-policy.md)
- [Not Waiting for Child Workflows to Start](../not-waiting-for-child-workflows-to-start.md)
- [Unnecessary Child Workflows](../unnecessary-child-workflows.md)
- [Child Workflow](child-workflow.md)
- [Terminate](terminate.md)
- [Cancellation](cancellation.md)
