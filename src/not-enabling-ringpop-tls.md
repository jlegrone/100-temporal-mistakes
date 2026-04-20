# Not Enabling Ringpop TLS

> [!TIP]
> Even if you have configured TLS for client-to-server and inter-service gRPC communication, Ringpop cluster membership traffic may still be unencrypted. Enable Ringpop TLS separately to fully secure all inter-node communication.

Temporal's server nodes form a cluster using Ringpop, a protocol based on SWIM (Scalable Weakly-consistent Infection-style Process Group Membership). Ringpop handles membership discovery and consistent hashing for routing requests to the correct node. It communicates over a dedicated port (typically the service's gRPC port + 1) using TCP, separate from gRPC inter-service communication with its own TLS configuration. Teams commonly set up TLS for the gRPC frontend and inter-service connections but overlook Ringpop, leaving cluster membership traffic unencrypted. Without Ringpop TLS, an attacker with network access could observe cluster topology, inject a rogue node into the membership ring, or disrupt cluster stability by sending crafted membership messages.

Configure Ringpop TLS in the Temporal server's YAML configuration, separately from gRPC TLS settings:

```yaml
global:
  membership:
    tls:
      enabled: true
      certFile: /path/to/ringpop-cert.pem
      keyFile: /path/to/ringpop-key.pem
      caFile: /path/to/ca-cert.pem
      requireClientAuth: true
```

Use dedicated certificates rather than reusing gRPC certificates for finer-grained control. Ringpop TLS is all-or-nothing within a cluster -- if some nodes have it enabled and others don't, they cannot communicate and the cluster will split. Plan for certificate rotation to avoid cluster downtime when certificates expire.

See also: [Not Setting Up TLS](not_setting_up_tls/) for securing client-to-server and inter-service gRPC connections.
