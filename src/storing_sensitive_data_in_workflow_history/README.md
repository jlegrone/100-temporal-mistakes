# Storing Sensitive Data in Workflow History

<!-- TODO: Note Temporal payload encryption and/or external storage as potential workarounds for this problem. Find an official temporal doc link for both approaches (https://github.com/temporalio/features/issues/765 is the issue for external payload storage, but there's probably a better doc link). -->

> [!TIP]
> [Workflow history](../terms/event-history.md) is persisted in the [Temporal server backend](../terms/temporal-server-backend.md) and visible through the UI and API. Sensitive data (PII, credentials, financial data) in history creates security and compliance risks.

Everything that flows through a Temporal workflow is persisted: workflow inputs and outputs, activity inputs and outputs, [signal](../terms/signals.md) and [update](../terms/updates.md) payloads, [search attributes](../terms/search-attributes.md), and memo fields. This data is accessible via the Temporal UI, CLI, and API to anyone with [namespace](../terms/namespace.md) permissions, which is often broader access than your production databases and could create a compliance risk.

The most robust approach is to keep sensitive data out of Temporal entirely: store it in a system with well-scoped access controls (a secrets manager, an encrypted database) and pass only references through workflows:

<!--SNIPSTART storing-sensitive-data-good-->
[storing_sensitive_data_in_workflow_history/client.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/storing_sensitive_data_in_workflow_history/client.go)
```go

// Good: pass a reference, not the data
type ProcessPaymentInputGood struct {
	PaymentTokenID string // Reference to payment details stored in a vault
	OrderID        string
	Amount         int64
}

```
<!--SNIPEND-->

<!--SNIPSTART storing-sensitive-data-bad-->
[storing_sensitive_data_in_workflow_history/client.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/storing_sensitive_data_in_workflow_history/client.go)
```go

// Bad: pass the sensitive data directly
type ProcessPaymentInputBad struct {
	CreditCardNumber string
	CVV              string
	OrderID          string
	Amount           int64
}

```
<!--SNIPEND-->

For defense in depth, a custom payload codec can be used which encrypts all payloads at rest or persists payloads in external storage outside of the Temporal server. Note that custom search attributes and semantic workflow IDs are never encrypted.
