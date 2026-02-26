# Storing Sensitive Data in Workflow History

> [!TIP]
> * [Workflow history](terms/event-history.md) -- including activity inputs/outputs, [signal](terms/signals.md) [payloads](terms/payload.md), and workflow arguments -- is persisted in the [Temporal server backend](terms/temporal-server-backend.md) and visible through the UI and API.
> * Sensitive data (PII, credentials, financial data) in history creates security and compliance risks.
> * Use a custom [data converter](terms/data-converter.md) with encryption, or store sensitive data externally and pass only references.

## What?

Everything that flows through a Temporal workflow is persisted: workflow inputs and outputs, activity inputs and outputs, signal and [update](terms/updates.md) payloads, [query](terms/queries.md) results, [search attributes](terms/search-attributes.md), and memo fields. All of this data is stored in the [Temporal server backend](terms/temporal-server-backend.md) database and is accessible via the Temporal UI, CLI, and API to anyone with the appropriate [namespace](terms/namespace.md) permissions.

The mistake is treating workflow history like an internal, private data store and passing sensitive information -- passwords, API keys, personally identifiable information, credit card numbers, health records -- directly as workflow or activity parameters.

## Why?

Sensitive data in workflow history creates multiple risk vectors:

- **Broad visibility.** Anyone with access to the Temporal namespace can inspect workflow histories. In many organizations, operations teams, developers, and on-call engineers all have access. This is far broader than the access controls on your production databases.
- **Persistence and retention.** Workflow histories are retained for the configured namespace retention period (default: 72 hours for closed workflows, indefinitely for open ones). Sensitive data lives in the database for that entire duration.
- **Compliance violations.** Regulations like GDPR, HIPAA, and PCI-DSS have strict requirements about where sensitive data can be stored, who can access it, and how long it's retained. Workflow history often falls outside the scope of your data governance controls.
- **Logging and debugging.** Temporal's tooling is designed to make workflow data easily inspectable for debugging purposes. This is a feature for operational data, but a liability for sensitive data.

## How?

### Use a custom data converter with encryption

Temporal SDKs support custom [data converters](terms/data-converter.md) that can encrypt payloads before they're sent to the server. The data is stored encrypted and only decrypted on [workers](terms/worker.md) that have the encryption key:

```go
// Configure the client with an encrypting data converter
client, err := client.Dial(client.Options{
    DataConverter: encryption.NewDataConverter(
        converter.GetDefaultDataConverter(),
        encryption.DataConverterOptions{KeyID: "my-key-id"},
    ),
})
```

With this approach, the Temporal server stores only encrypted blobs. The UI shows encrypted data unless configured with a codec server that can decrypt it.

### Store sensitive data externally

The most robust approach is to keep sensitive data out of Temporal entirely. Store it in a system with proper access controls (a secrets manager, an encrypted database) and pass only references through workflows:

```go
// Good: pass a reference, not the data
type ProcessPaymentInput struct {
    PaymentTokenID string  // Reference to payment details stored in a vault
    OrderID        string
    Amount         int64
}

// Bad: pass the sensitive data directly
type ProcessPaymentInput struct {
    CreditCardNumber string
    CVV              string
    OrderID          string
    Amount           int64
}
```

### Combine both approaches

For defense in depth, use encryption at the data converter level and also minimize the sensitive data that passes through Temporal. Encryption protects against database breaches and unauthorized access, while minimizing data reduces the blast radius if encryption is ever compromised.
