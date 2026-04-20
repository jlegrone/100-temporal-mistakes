# Using the Temporal UI for Non-Debugging Purposes

> [!TIP]
> The Temporal Web UI is an operator debugging tool, not an application interface. Build proper API layers for workflow interactions instead of having end users send signals or start workflows through the UI.

## What?

The Temporal Web UI is a powerful debugging tool. It lets you inspect [workflow histories](terms/event-history.md), view workflow state via [queries](terms/queries.md), send [signals](terms/signals.md), and start new workflow executions. Because it's so capable, teams sometimes use it as the primary interface for their workflows -- having end users or operations staff trigger actions through the UI instead of building proper application interfaces.

The UI is an operator tool, not an application layer. When you embed it into your business process ("when an order needs approval, go to the Temporal UI and send a signal to the workflow"), you've coupled your business process to a debugging tool.

## Why?

**No access control at the workflow level.** The Temporal UI provides [namespace](terms/namespace.md)-level access control, not workflow-level. You can't say "this user can only signal their own orders." Anyone with access to the namespace can interact with any workflow in it.

**No input validation.** When you send a signal or start a workflow from the UI, you provide raw JSON [payloads](terms/payload.md). There's no schema validation, no business rule enforcement, no guard rails. One malformed payload and your workflow receives garbage data.

**No auditability from your application's perspective.** While Temporal records the history event, your application has no record of who did what and why. You lose the audit trail that a proper application layer would provide.

**It doesn't scale.** Training end users to navigate the Temporal UI, find the right [workflow ID](terms/workflow-id.md), and send correctly formatted signals is error-prone and slow. It works for a handful of workflows during development but falls apart in production.

**UI changes are outside your control.** The Temporal UI evolves with the platform. Layouts change, features move, endpoints get updated. If your business process depends on specific UI interactions, you're at the mercy of upstream changes.

## Solution

**Build application APIs that wrap Temporal interactions.** Your application should have its own endpoints for workflow actions. These endpoints handle authentication, authorization, input validation, and audit logging before making the corresponding Temporal SDK call.

**Use the Temporal UI for what it's good at.** It excels at debugging stuck workflows, inspecting event histories, and understanding what happened during an execution. Keep it as an operator tool, not a user tool.

**For human-in-the-loop workflows**, build dedicated UIs. If your workflow needs human approval or input, build an application screen for that specific interaction. The screen calls your API, which sends the signal or update to the workflow. The end user never needs to know Temporal exists.

**Restrict UI access to operators and developers.** The Temporal UI should be accessible to people who need to debug and operate workflows, not to end users interacting with your product.
