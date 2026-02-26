# Using the Temporal UI for Non-Debugging Purposes

> [!TIP]
> * The Temporal Web UI is designed for debugging and operational visibility, not as an application interface.
> * Building user-facing workflows around manual UI interactions (triggering [signals](terms/signals.md), [updates](terms/updates.md), or starting workflows from the UI) creates a fragile, non-scalable system.
> * Build proper API layers and application UIs that interact with Temporal programmatically.

## What?

The Temporal Web UI is a powerful debugging tool. It lets you inspect workflow histories, view workflow state via [queries](terms/queries.md), send [signals](terms/signals.md), and even start new workflow executions. Because it's so capable, teams sometimes start using it as the primary interface for interacting with their workflows -- having end users or operations staff trigger actions through the UI instead of building proper application interfaces.

This is a mistake. The UI is an operator tool, not an application layer. When you embed it into your business process ("when an order needs approval, go to the Temporal UI and send a signal to the workflow"), you've coupled your business process to a debugging tool.

## Why?

**No access control at the workflow level.** The Temporal UI provides namespace-level access control, not workflow-level. You can't say "this user can only signal their own orders." Anyone with access to the namespace can interact with any workflow in it.

**No input validation.** When you send a signal or start a workflow from the UI, you're providing raw JSON payloads. There's no schema validation, no business rule enforcement, no guard rails. One malformed payload and your workflow receives garbage data.

**No auditability from your application's perspective.** While Temporal records the history event, your application has no record of who did what and why. You lose the audit trail that a proper application layer would provide.

**It doesn't scale as a process.** Training end users to navigate the Temporal UI, find the right workflow ID, and send correctly formatted signals is error-prone and slow. It works for a handful of workflows during development but falls apart at production scale.

**UI changes are outside your control.** The Temporal UI evolves with the platform. Layouts change, features move, and endpoints get updated. If your business process depends on specific UI interactions, you're at the mercy of upstream changes.

## Solution

**Build application APIs that wrap Temporal interactions.** Your application should have its own endpoints for triggering workflow actions. These endpoints handle authentication, authorization, input validation, and audit logging before making the corresponding Temporal SDK call.

**Use the Temporal UI for what it's good at.** It excels at debugging stuck workflows, inspecting event histories, and understanding what happened during an execution. Keep it as an operator tool, not a user tool.

**For human-in-the-loop workflows**, build dedicated UIs. If your workflow needs human approval or input, build an application screen for that specific interaction. The screen calls your API, which sends the signal or update to the workflow. The end user never needs to know Temporal exists.

**Restrict UI access to operators and developers.** The Temporal UI should be accessible to people who need to debug and operate workflows, not to end users interacting with your product.
