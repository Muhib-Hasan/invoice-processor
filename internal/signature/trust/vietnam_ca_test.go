package trust

import (
	"testing"
	"time"
)

func TestLoadVietnamRootCAs(t *testing.T) {
	pool, err := LoadVietnamRootCAs()
	if err != nil {
		t.Fatalf("LoadVietnamRootCAs failed: %v", err)
	}

	if pool == nil {
		t.Fatal("pool should not be nil")
	}

	// Pool should have at least 2 certs (G2 + G3)
	// Note: x509.CertPool doesn't expose count, but we can verify loading worked
}

func TestLoadVietnamTSACert(t *testing.T) {
	cert, err := LoadVietnamTSACert()
	if err != nil {
		t.Fatalf("LoadVietnamTSACert failed: %v", err)
	}

	if cert == nil {
		t.Fatal("cert should not be nil")
	}

	// Verify it's the TSA cert
	if cert.Subject.CommonName != "VIETNAM NATIONAL ROOT CA - TSA" {
		t.Errorf("unexpected CN: %s", cert.Subject.CommonName)
	}

	// Verify validity period
	expectedNotBefore := time.Date(2025, 11, 25, 0, 0, 0, 0, time.UTC)
	if cert.NotBefore.Before(expectedNotBefore) {
		t.Errorf("TSA cert NotBefore unexpected: %v", cert.NotBefore)
	}
}

func TestLoadAllVietnamCerts(t *testing.T) {
	pool, err := LoadAllVietnamCerts()
	if err != nil {
		t.Fatalf("LoadAllVietnamCerts failed: %v", err)
	}

	if pool == nil {
		t.Fatal("pool should not be nil")
	}
}

func TestGetVietnamRootCAInfo(t *testing.T) {
	info := GetVietnamRootCAInfo()

	if len(info) != 3 {
		t.Errorf("expected 3 cert infos, got %d", len(info))
	}

	// Verify G3 info
	foundG3 := false
	for _, ci := range info {
		if ci.Name == "Vietnam National Root CA G3" {
			foundG3 = true
			if ci.Validity != "2024-2049" {
				t.Errorf("G3 validity: got %s, want 2024-2049", ci.Validity)
			}
		}
	}
	if !foundG3 {
		t.Error("G3 cert info not found")
	}
}

func TestLoadEmbeddedCert_G2(t *testing.T) {
	cert, err := loadEmbeddedCert(certPathG2)
	if err != nil {
		t.Fatalf("loadEmbeddedCert(G2) failed: %v", err)
	}

	if cert.Subject.CommonName != "Vietnam National Root CA" {
		t.Errorf("G2 CN: got %s", cert.Subject.CommonName)
	}

	// Verify it's a CA certificate
	if !cert.IsCA {
		t.Error("G2 should be a CA certificate")
	}
}

func TestLoadEmbeddedCert_G3(t *testing.T) {
	cert, err := loadEmbeddedCert(certPathG3)
	if err != nil {
		t.Fatalf("loadEmbeddedCert(G3) failed: %v", err)
	}

	if cert.Subject.CommonName != "Vietnam National Root CA G3" {
		t.Errorf("G3 CN: got %s", cert.Subject.CommonName)
	}

	// Verify it's a CA certificate
	if !cert.IsCA {
		t.Error("G3 should be a CA certificate")
	}
}

func TestLoadEmbeddedCert_InvalidPath(t *testing.T) {
	_, err := loadEmbeddedCert("certs/nonexistent.crt")
	if err == nil {
		t.Error("expected error for nonexistent cert")
	}
}
