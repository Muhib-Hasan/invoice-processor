package trust

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// TrustStore manages trusted CA certificates and revocation checking
type TrustStore struct {
	roots       *x509.CertPool
	rootCerts   []*x509.Certificate // Keep track of individual certs for libraries that need slice
	ocspCache   *OCSPCache
	ocspTimeout time.Duration
	softFail    bool
}

// TrustStoreOption configures a TrustStore
type TrustStoreOption func(*TrustStore)

// NewTrustStore creates a new trust store with Vietnam root CAs by default
func NewTrustStore(opts ...TrustStoreOption) (*TrustStore, error) {
	roots, err := LoadVietnamRootCAs()
	if err != nil {
		return nil, fmt.Errorf("failed to load Vietnam root CAs: %w", err)
	}

	rootCerts, err := LoadVietnamRootCAsSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to load Vietnam root CAs slice: %w", err)
	}

	store := &TrustStore{
		roots:       roots,
		rootCerts:   rootCerts,
		ocspCache:   NewOCSPCache(DefaultOCSPCacheTTL),
		ocspTimeout: DefaultOCSPTimeout,
		softFail:    false,
	}

	for _, opt := range opts {
		opt(store)
	}

	return store, nil
}

// NewEmptyTrustStore creates a trust store without default CAs
func NewEmptyTrustStore(opts ...TrustStoreOption) *TrustStore {
	store := &TrustStore{
		roots:       x509.NewCertPool(),
		rootCerts:   make([]*x509.Certificate, 0),
		ocspCache:   NewOCSPCache(DefaultOCSPCacheTTL),
		ocspTimeout: DefaultOCSPTimeout,
		softFail:    false,
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// WithSoftFail enables soft-fail mode for OCSP checks
// When enabled, OCSP failures don't cause verification to fail
func WithSoftFail() TrustStoreOption {
	return func(s *TrustStore) {
		s.softFail = true
	}
}

// WithOCSPTimeout sets the timeout for OCSP requests
func WithOCSPTimeout(d time.Duration) TrustStoreOption {
	return func(s *TrustStore) {
		s.ocspTimeout = d
	}
}

// WithOCSPCacheTTL sets the TTL for OCSP cache entries
func WithOCSPCacheTTL(d time.Duration) TrustStoreOption {
	return func(s *TrustStore) {
		s.ocspCache = NewOCSPCache(d)
	}
}

// WithCustomCertsFromFile adds custom CA certificates from a PEM file
func WithCustomCertsFromFile(path string) TrustStoreOption {
	return func(s *TrustStore) {
		data, err := os.ReadFile(path)
		if err != nil {
			return // silently ignore, can check with HasCustomCerts
		}
		s.AddCertificatesFromPEM(data)
	}
}

// AddCertificate adds a single certificate to the trust store
func (s *TrustStore) AddCertificate(cert *x509.Certificate) {
	if cert != nil {
		s.roots.AddCert(cert)
		s.rootCerts = append(s.rootCerts, cert)
	}
}

// AddCertificates adds multiple certificates to the trust store
func (s *TrustStore) AddCertificates(certs ...*x509.Certificate) {
	for _, cert := range certs {
		s.AddCertificate(cert)
	}
}

// AddCertificatesFromPEM parses and adds certificates from PEM data
func (s *TrustStore) AddCertificatesFromPEM(pemData []byte) error {
	var added int
	for {
		block, rest := pem.Decode(pemData)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse certificate: %w", err)
			}
			s.roots.AddCert(cert)
			added++
		}
		pemData = rest
	}
	if added == 0 {
		return fmt.Errorf("no certificates found in PEM data")
	}
	return nil
}

// VerifyChain verifies the certificate chain against trusted roots
func (s *TrustStore) VerifyChain(cert *x509.Certificate, intermediates []*x509.Certificate) ([]*x509.Certificate, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}

	// Build intermediate pool
	var interPool *x509.CertPool
	if len(intermediates) > 0 {
		interPool = x509.NewCertPool()
		for _, inter := range intermediates {
			interPool.AddCert(inter)
		}
	}

	opts := x509.VerifyOptions{
		Roots:         s.roots,
		Intermediates: interPool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("chain verification failed: %w", err)
	}

	if len(chains) == 0 {
		return nil, fmt.Errorf("no valid certificate chains found")
	}

	return chains[0], nil
}

// CheckRevocation checks if a certificate has been revoked using OCSP
func (s *TrustStore) CheckRevocation(ctx context.Context, cert *x509.Certificate, issuer *x509.Certificate) (bool, error) {
	if cert == nil || issuer == nil {
		return false, fmt.Errorf("certificate or issuer is nil")
	}

	// Check cache first
	if revoked, found := s.ocspCache.Get(cert); found {
		return !revoked, nil
	}

	// No OCSP URLs - assume not revoked but warn
	if len(cert.OCSPServer) == 0 {
		return true, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.ocspTimeout)
	defer cancel()

	// Perform OCSP check
	revoked, err := CheckOCSP(ctx, cert, issuer)
	if err != nil {
		if s.softFail {
			// Soft-fail: assume not revoked
			return true, fmt.Errorf("OCSP check failed (soft-fail enabled): %w", err)
		}
		return false, fmt.Errorf("OCSP check failed: %w", err)
	}

	// Cache result
	s.ocspCache.Set(cert, !revoked)

	return !revoked, nil
}

// Roots returns the certificate pool
func (s *TrustStore) Roots() *x509.CertPool {
	return s.roots
}

// RootCerts returns the root certificates as a slice
// Useful for libraries that require []*x509.Certificate instead of *x509.CertPool
func (s *TrustStore) RootCerts() []*x509.Certificate {
	return s.rootCerts
}

// IsSoftFail returns whether soft-fail mode is enabled
func (s *TrustStore) IsSoftFail() bool {
	return s.softFail
}
