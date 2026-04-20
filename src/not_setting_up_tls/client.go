package not_setting_up_tls

import (
	"crypto/tls"
	"crypto/x509"

	"go.temporal.io/sdk/client"
)

// certPool and clientCert are placeholders for actual TLS credentials.
var (
	certPool   *x509.CertPool
	clientCert tls.Certificate
)

// @@@SNIPSTART not-setting-up-tls-good

// Good: configure TLS on all SDK clients.
var clientOptions = client.Options{
	HostPort: "temporal.example.com:7233",
	ConnectionOptions: client.ConnectionOptions{
		TLS: &tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{clientCert},
		},
	},
}

// @@@SNIPEND
