package trust

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func TestOCSPCache_GetSet(t *testing.T) {
	cache := NewOCSPCache(time.Hour)

	cert := createTestCertForCache(t, "Test Cert")

	// Initially not found
	_, found := cache.Get(cert)
	if found {
		t.Error("expected not found for new cert")
	}

	// Set and get
	cache.Set(cert, true) // not revoked
	notRevoked, found := cache.Get(cert)
	if !found {
		t.Error("expected found after Set")
	}
	if !notRevoked {
		t.Error("expected notRevoked=true")
	}

	// Set revoked
	cache.Set(cert, false)
	notRevoked, found = cache.Get(cert)
	if !found {
		t.Error("expected found after second Set")
	}
	if notRevoked {
		t.Error("expected notRevoked=false")
	}
}

func TestOCSPCache_Expiration(t *testing.T) {
	// Very short TTL for testing
	cache := NewOCSPCache(10 * time.Millisecond)

	cert := createTestCertForCache(t, "Test Cert")

	cache.Set(cert, true)

	// Should be found immediately
	_, found := cache.Get(cert)
	if !found {
		t.Error("expected found immediately after Set")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be expired now
	_, found = cache.Get(cert)
	if found {
		t.Error("expected not found after expiration")
	}
}

func TestOCSPCache_Clear(t *testing.T) {
	cache := NewOCSPCache(time.Hour)

	cert1 := createTestCertForCache(t, "Cert 1")
	cert2 := createTestCertForCache(t, "Cert 2")

	cache.Set(cert1, true)
	cache.Set(cert2, false)

	if cache.Size() != 2 {
		t.Errorf("size: got %d, want 2", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("size after clear: got %d, want 0", cache.Size())
	}

	_, found := cache.Get(cert1)
	if found {
		t.Error("expected not found after Clear")
	}
}

func TestOCSPCache_NilCert(t *testing.T) {
	cache := NewOCSPCache(time.Hour)

	// Get with nil should return not found
	_, found := cache.Get(nil)
	if found {
		t.Error("expected not found for nil cert")
	}

	// Set with nil should not panic
	cache.Set(nil, true) // should be a no-op
}

func TestOCSPCache_DifferentCerts(t *testing.T) {
	cache := NewOCSPCache(time.Hour)

	cert1 := createTestCertForCache(t, "Cert 1")
	cert2 := createTestCertForCache(t, "Cert 2")

	cache.Set(cert1, true)
	cache.Set(cert2, false)

	// Verify they don't interfere
	notRevoked1, found1 := cache.Get(cert1)
	notRevoked2, found2 := cache.Get(cert2)

	if !found1 || !found2 {
		t.Error("expected both certs to be found")
	}
	if !notRevoked1 {
		t.Error("cert1 should be not revoked")
	}
	if notRevoked2 {
		t.Error("cert2 should be revoked")
	}
}

func TestCertCacheKey(t *testing.T) {
	cert := createTestCertForCache(t, "Test")
	key := certCacheKey(cert)

	if key == "" {
		t.Error("cache key should not be empty")
	}

	// Same cert should produce same key
	key2 := certCacheKey(cert)
	if key != key2 {
		t.Error("same cert should produce same key")
	}
}

// Helper function
func createTestCertForCache(t *testing.T, cn string) *x509.Certificate {
	t.Helper()

	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: cn,
		},
		Issuer: pkix.Name{
			CommonName: "Test Issuer",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	cert, _ := x509.ParseCertificate(certDER)
	return cert
}
