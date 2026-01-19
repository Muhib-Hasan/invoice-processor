package trust

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/ocsp"
)

// Default OCSP configuration
const (
	DefaultOCSPTimeout  = 10 * time.Second
	DefaultOCSPCacheTTL = 1 * time.Hour
)

// OCSPCache caches OCSP responses to reduce latency
type OCSPCache struct {
	mu      sync.RWMutex
	entries map[string]*ocspCacheEntry
	ttl     time.Duration
}

type ocspCacheEntry struct {
	notRevoked bool
	expiresAt  time.Time
}

// NewOCSPCache creates a new OCSP response cache
func NewOCSPCache(ttl time.Duration) *OCSPCache {
	return &OCSPCache{
		entries: make(map[string]*ocspCacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached OCSP result
func (c *OCSPCache) Get(cert *x509.Certificate) (notRevoked bool, found bool) {
	if cert == nil {
		return false, false
	}

	key := certCacheKey(cert)

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return false, false
	}

	if time.Now().After(entry.expiresAt) {
		// Entry expired, remove it
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return false, false
	}

	return entry.notRevoked, true
}

// Set caches an OCSP result
func (c *OCSPCache) Set(cert *x509.Certificate, notRevoked bool) {
	if cert == nil {
		return
	}

	key := certCacheKey(cert)

	c.mu.Lock()
	c.entries[key] = &ocspCacheEntry{
		notRevoked: notRevoked,
		expiresAt:  time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Clear removes all cached entries
func (c *OCSPCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*ocspCacheEntry)
	c.mu.Unlock()
}

// Size returns the number of cached entries
func (c *OCSPCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// certCacheKey generates a cache key from certificate serial number and issuer
func certCacheKey(cert *x509.Certificate) string {
	return fmt.Sprintf("%s:%s", cert.Issuer.String(), cert.SerialNumber.String())
}

// CheckOCSP performs an OCSP check for a certificate
func CheckOCSP(ctx context.Context, cert, issuer *x509.Certificate) (revoked bool, err error) {
	if len(cert.OCSPServer) == 0 {
		return false, fmt.Errorf("no OCSP server URL in certificate")
	}

	// Create OCSP request
	ocspRequest, err := ocsp.CreateRequest(cert, issuer, &ocsp.RequestOptions{
		Hash: crypto.SHA256,
	})
	if err != nil {
		return false, fmt.Errorf("failed to create OCSP request: %w", err)
	}

	// Try each OCSP server
	var lastErr error
	for _, server := range cert.OCSPServer {
		revoked, err := queryOCSPServer(ctx, server, ocspRequest, issuer)
		if err == nil {
			return revoked, nil
		}
		lastErr = err
	}

	return false, fmt.Errorf("all OCSP servers failed: %w", lastErr)
}

// queryOCSPServer sends an OCSP request to a specific server
func queryOCSPServer(ctx context.Context, serverURL string, request []byte, issuer *x509.Certificate) (revoked bool, err error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(request))
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/ocsp-request")
	req.Header.Set("Accept", "application/ocsp-response")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("OCSP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("OCSP server returned status %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read OCSP response: %w", err)
	}

	// Parse response
	ocspResp, err := ocsp.ParseResponseForCert(body, nil, issuer)
	if err != nil {
		return false, fmt.Errorf("failed to parse OCSP response: %w", err)
	}

	// Check status
	switch ocspResp.Status {
	case ocsp.Good:
		return false, nil
	case ocsp.Revoked:
		return true, nil
	case ocsp.Unknown:
		return false, fmt.Errorf("OCSP status unknown")
	default:
		return false, fmt.Errorf("unexpected OCSP status: %d", ocspResp.Status)
	}
}
