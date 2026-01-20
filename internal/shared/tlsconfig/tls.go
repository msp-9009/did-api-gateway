package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Config holds TLS configuration
type Config struct {
	// Server TLS
	CertFile string
	KeyFile  string
	
	// Client TLS (for mTLS)
	ClientCAFile string
	RequireClientCert bool
	
	// Security settings
	MinVersion         uint16
	CipherSuites       []uint16
	PreferServerCipher bool
}

// LoadServerTLSConfig creates a TLS config for HTTPS servers
func LoadServerTLSConfig(cfg Config) (*tls.Config, error) {
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("cert file and key file are required")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   cfg.MinVersion,
		CipherSuites: cfg.CipherSuites,
		PreferServerCipherSuites: cfg.PreferServerCipher,
	}

	// Set secure defaults if not specified
	if tlsConfig.MinVersion == 0 {
		tlsConfig.MinVersion = tls.VersionTLS13
	}

	if len(tlsConfig.CipherSuites) == 0 {
		// Use secure cipher suites (TLS 1.3 ciphers are always enabled)
		tlsConfig.CipherSuites = []uint16{
			// TLS 1.3 suites (used automatically)
			// TLS 1.2 suites for backward compatibility
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		}
	}

	// Configure mTLS if client CA is provided
	if cfg.ClientCAFile != "" {
		clientCAPool := x509.NewCertPool()
		clientCAPEM, err := os.ReadFile(cfg.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read client CA file: %w", err)
		}
		if !clientCAPool.AppendCertsFromPEM(clientCAPEM) {
			return nil, fmt.Errorf("failed to parse client CA certificate")
		}

		tlsConfig.ClientCAs = clientCAPool
		if cfg.RequireClientCert {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
		}
	}

	return tlsConfig, nil
}

// LoadClientTLSConfig creates a TLS config for HTTPS clients (reverse proxy, DID resolution)
func LoadClientTLSConfig(serverCAFile string, clientCertFile string, clientKeyFile string) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// Load server CA for verification (if not using system CAs)
	if serverCAFile != "" {
		serverCAPool := x509.NewCertPool()
		serverCAPEM, err := os.ReadFile(serverCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read server CA file: %w", err)
		}
		if !serverCAPool.AppendCertsFromPEM(serverCAPEM) {
			return nil, fmt.Errorf("failed to parse server CA certificate")
		}
		tlsConfig.RootCAs = serverCAPool
	}

	// Load client certificate for mTLS
	if clientCertFile != "" && clientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// GenerateSelfSignedCert generates a self-signed certificate for local development
// This should only be used for development, never in production
func GenerateSelfSignedCert(certFile, keyFile string, hosts []string) error {
	// This is a placeholder - implementation would use crypto/x509
	// For actual implementation, use a library or script
	return fmt.Errorf("use openssl or mkcert to generate self-signed certificates for development")
}
