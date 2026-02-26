# Not Setting Up TLS

> [!TIP]
> * By default, all Temporal SDK-to-server communication is unencrypted, meaning workflow data travels in plaintext over the network.
> * In any production environment, TLS must be configured for [worker](terms/worker.md)-to-server and client-to-server connections.
> * Temporal supports mutual TLS (mTLS) for both authentication and encryption, which you should enable when possible.

## What?

Temporal SDKs communicate with the Temporal server over gRPC. Out of the box, these connections are unencrypted. This means all data flowing between your workers and the server (workflow inputs, activity outputs, [heartbeat](terms/heartbeat.md) [payloads](terms/payload.md), etc.) is transmitted in plaintext. Anyone with network access can observe or tamper with this traffic.

This applies to all SDK connections: workers polling for tasks, clients starting workflows, and clients querying workflow state.

## Why?

Workflow payloads often contain sensitive business data: customer information, financial records, internal identifiers, and so on. Even if you are using a custom [data converter](terms/data-converter.md) to encrypt payloads at the application level, metadata such as [workflow IDs](terms/workflow-id.md), [task queue](terms/task-queue.md) names, [namespace](terms/namespace.md) names, and Temporal headers are still transmitted in the clear without TLS.

Beyond confidentiality, TLS also provides integrity (protection against tampering) and, with mutual TLS, authentication (verifying the identity of both client and server). Without mTLS, any process that can reach the Temporal server's gRPC port can start workflows, send [signals](terms/signals.md), or [terminate](terms/terminate.md) running workflows.

In production environments, running without TLS typically violates security compliance requirements (SOC 2, HIPAA, PCI-DSS, etc.) regardless of whether the traffic stays within a private network.

## How?

**Configure TLS on the Temporal server.** The server configuration supports specifying certificate and key files for its frontend gRPC service. At a minimum, provide a server certificate and key.

**Configure TLS on all SDK clients.** Each SDK provides connection options for specifying TLS settings. In Go:

```go
clientOptions := client.Options{
    HostPort:  "temporal.example.com:7233",
    ConnectionOptions: client.ConnectionOptions{
        TLS: &tls.Config{
            // Server CA certificate to verify the server
            RootCAs: certPool,
            // Client certificate for mutual TLS
            Certificates: []tls.Certificate{clientCert},
        },
    },
}
```

**Use mutual TLS (mTLS) when possible.** mTLS adds client authentication on top of encryption. The server verifies the client's certificate, ensuring only authorized workers and clients can connect. This is especially important if your Temporal server is accessible from multiple networks or if you want to enforce per-namespace access controls based on client certificates.

**Don't forget inter-service communication.** If you are self-hosting Temporal, the server consists of multiple internal services (frontend, history, matching, worker) that communicate with each other. TLS should also be configured for these inter-service connections. See also [Not enabling Ringpop TLS](not-enabling-ringpop-tls.md) for the often-overlooked cluster membership protocol.
