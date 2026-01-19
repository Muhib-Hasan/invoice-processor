package trust

import (
	"crypto/x509"
	"embed"
	"encoding/pem"
	"fmt"
)

// Embedded Vietnam National Root CA certificates
//
//go:embed certs/vietnam-nrca-sha256-g3.crt
//go:embed certs/vietnam-nrca-sha256-g2.crt
//go:embed certs/vietnam-nrca-tsa.crt
var vietnamRootCerts embed.FS

// Certificate file paths
const (
	certPathG3  = "certs/vietnam-nrca-sha256-g3.crt"
	certPathG2  = "certs/vietnam-nrca-sha256-g2.crt"
	certPathTSA = "certs/vietnam-nrca-tsa.crt"
)

// LoadVietnamRootCAs loads Vietnam National Root CA certificates (G2 + G3)
// These are the root certificates that all Vietnam e-invoice signing CAs chain to
func LoadVietnamRootCAs() (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	// Load G3 (current root, 2024-2049)
	g3Cert, err := loadEmbeddedCert(certPathG3)
	if err != nil {
		return nil, fmt.Errorf("failed to load G3 cert: %w", err)
	}
	pool.AddCert(g3Cert)

	// Load G2 (legacy root, 2014-2039)
	g2Cert, err := loadEmbeddedCert(certPathG2)
	if err != nil {
		return nil, fmt.Errorf("failed to load G2 cert: %w", err)
	}
	pool.AddCert(g2Cert)

	return pool, nil
}

// LoadVietnamTSACert loads the Vietnam National Root CA TSA certificate
// Used for timestamp verification
func LoadVietnamTSACert() (*x509.Certificate, error) {
	return loadEmbeddedCert(certPathTSA)
}

// LoadAllVietnamCerts loads all Vietnam root certificates including TSA
func LoadAllVietnamCerts() (*x509.CertPool, error) {
	pool, err := LoadVietnamRootCAs()
	if err != nil {
		return nil, err
	}

	tsaCert, err := LoadVietnamTSACert()
	if err != nil {
		return nil, fmt.Errorf("failed to load TSA cert: %w", err)
	}
	pool.AddCert(tsaCert)

	return pool, nil
}

// loadEmbeddedCert loads a certificate from embedded FS
func loadEmbeddedCert(path string) (*x509.Certificate, error) {
	data, err := vietnamRootCerts.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded cert %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %w", path, err)
	}

	return cert, nil
}

// GetVietnamRootCAInfo returns information about the embedded Vietnam root CAs
func GetVietnamRootCAInfo() []CertInfo {
	return []CertInfo{
		{
			Name:     "Vietnam National Root CA G3",
			File:     certPathG3,
			Validity: "2024-2049",
			Purpose:  "Current root for new e-invoices",
		},
		{
			Name:     "Vietnam National Root CA (SHA-256)",
			File:     certPathG2,
			Validity: "2014-2039",
			Purpose:  "Legacy root for older e-invoices",
		},
		{
			Name:     "Vietnam National Root CA - TSA",
			File:     certPathTSA,
			Validity: "2025-2050",
			Purpose:  "Timestamp verification",
		},
	}
}

// CertInfo contains metadata about an embedded certificate
type CertInfo struct {
	Name     string
	File     string
	Validity string
	Purpose  string
}

// LoadVietnamRootCAsSlice loads Vietnam National Root CA certificates as a slice
// Useful for libraries that require []*x509.Certificate instead of *x509.CertPool
func LoadVietnamRootCAsSlice() ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0, 2)

	// Load G3 (current root, 2024-2049)
	g3Cert, err := loadEmbeddedCert(certPathG3)
	if err != nil {
		return nil, fmt.Errorf("failed to load G3 cert: %w", err)
	}
	certs = append(certs, g3Cert)

	// Load G2 (legacy root, 2014-2039)
	g2Cert, err := loadEmbeddedCert(certPathG2)
	if err != nil {
		return nil, fmt.Errorf("failed to load G2 cert: %w", err)
	}
	certs = append(certs, g2Cert)

	return certs, nil
}
