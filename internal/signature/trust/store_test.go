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

func TestNewTrustStore(t *testing.T) {
	store, err := NewTrustStore()
	if err != nil {
		t.Fatalf("NewTrustStore failed: %v", err)
	}

	if store.roots == nil {
		t.Error("roots should not be nil")
	}

	if store.ocspCache == nil {
		t.Error("ocspCache should not be nil")
	}

	if store.softFail {
		t.Error("softFail should be false by default")
	}
}

func TestNewTrustStore_WithOptions(t *testing.T) {
	store, err := NewTrustStore(
		WithSoftFail(),
		WithOCSPTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("NewTrustStore failed: %v", err)
	}

	if !store.softFail {
		t.Error("softFail should be true after WithSoftFail()")
	}

	if store.ocspTimeout != 5*time.Second {
		t.Errorf("ocspTimeout: got %v, want 5s", store.ocspTimeout)
	}
}

func TestNewEmptyTrustStore(t *testing.T) {
	store := NewEmptyTrustStore()

	if store.roots == nil {
		t.Error("roots should not be nil")
	}
}

func TestTrustStore_AddCertificatesFromPEM(t *testing.T) {
	store := NewEmptyTrustStore()

	// Create a test certificate
	cert, _ := createTestCert(t, "Test CA", true)
	pemData := certToPEM(cert)

	err := store.AddCertificatesFromPEM(pemData)
	if err != nil {
		t.Fatalf("AddCertificatesFromPEM failed: %v", err)
	}
}

func TestTrustStore_AddCertificatesFromPEM_Invalid(t *testing.T) {
	store := NewEmptyTrustStore()

	err := store.AddCertificatesFromPEM([]byte("not a certificate"))
	if err == nil {
		t.Error("expected error for invalid PEM data")
	}
}

func TestTrustStore_VerifyChain(t *testing.T) {
	// Create a root CA
	rootKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test Root CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, _ := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	rootCert, _ := x509.ParseCertificate(rootDER)

	// Create an end-entity certificate signed by root
	eeKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	eeTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "End Entity",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}
	eeDER, _ := x509.CreateCertificate(rand.Reader, eeTemplate, rootTemplate, &eeKey.PublicKey, rootKey)
	eeCert, _ := x509.ParseCertificate(eeDER)

	// Create trust store with root CA
	store := NewEmptyTrustStore()
	store.AddCertificate(rootCert)

	// Verify chain
	chain, err := store.VerifyChain(eeCert, nil)
	if err != nil {
		t.Fatalf("VerifyChain failed: %v", err)
	}

	if len(chain) != 2 {
		t.Errorf("chain length: got %d, want 2", len(chain))
	}
}

func TestTrustStore_VerifyChain_Untrusted(t *testing.T) {
	// Create an end-entity certificate with self-signed root
	rootKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Untrusted Root",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	rootDER, _ := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	rootCert, _ := x509.ParseCertificate(rootDER)

	// Create end-entity
	eeKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	eeTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "End Entity",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
	}
	eeDER, _ := x509.CreateCertificate(rand.Reader, eeTemplate, rootTemplate, &eeKey.PublicKey, rootKey)
	eeCert, _ := x509.ParseCertificate(eeDER)

	// Create empty trust store (no trusted roots)
	store := NewEmptyTrustStore()

	// Verify should fail - root not trusted
	_, err := store.VerifyChain(eeCert, []*x509.Certificate{rootCert})
	if err == nil {
		t.Error("expected error for untrusted root")
	}
}

func TestTrustStore_VerifyChain_NilCert(t *testing.T) {
	store := NewEmptyTrustStore()

	_, err := store.VerifyChain(nil, nil)
	if err == nil {
		t.Error("expected error for nil certificate")
	}
}

// Helper functions

func createTestCert(t *testing.T, cn string, isCA bool) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
	}

	if isCA {
		template.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, key
}

func certToPEM(cert *x509.Certificate) []byte {
	return []byte("-----BEGIN CERTIFICATE-----\n" +
		base64Encode(cert.Raw) +
		"\n-----END CERTIFICATE-----\n")
}

func base64Encode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, 0, (len(data)+2)/3*4)

	for i := 0; i < len(data); i += 3 {
		var val uint32
		remaining := len(data) - i
		if remaining >= 3 {
			val = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result = append(result, alphabet[val>>18&0x3F], alphabet[val>>12&0x3F], alphabet[val>>6&0x3F], alphabet[val&0x3F])
		} else if remaining == 2 {
			val = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result = append(result, alphabet[val>>18&0x3F], alphabet[val>>12&0x3F], alphabet[val>>6&0x3F], '=')
		} else {
			val = uint32(data[i]) << 16
			result = append(result, alphabet[val>>18&0x3F], alphabet[val>>12&0x3F], '=', '=')
		}
	}
	return string(result)
}
