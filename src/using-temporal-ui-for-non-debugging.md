# Using the Temporal UI for Non-Debugging Purposes
<!-- TODO: Delete this mistake from the repo. -->

> [!TIP]
> The Temporal Web UI is an operator debugging tool, not an application interface. Build proper API layers for workflow interactions instead of having end users send signals or start workflows through the UI.

The Temporal Web UI lets you inspect [workflow histories](terms/event-history.md), view state via [queries](terms/queries.md), send [signals](terms/signals.md), and start executions. Because it's so capable, teams sometimes use it as the primary interface -- having end users or operations staff trigger actions through the UI instead of building proper application interfaces. When you embed it into your business process ("when an order needs approval, go to the Temporal UI and send a signal"), you've coupled your business process to a debugging tool.

The UI provides [namespace](terms/namespace.md)-level access control, not workflow-level -- you can't restrict users to their own workflows. There's no input validation on raw JSON [payloads](terms/payload.md), no application-level audit trail, and the UI evolves with the platform outside your control.

Build application APIs that wrap Temporal interactions, handling authentication, authorization, input validation, and audit logging before making SDK calls. For human-in-the-loop workflows, build dedicated application screens -- the end user never needs to know Temporal exists. Keep the UI for what it's good at: debugging stuck workflows and inspecting event histories.
