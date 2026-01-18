package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/server"
)

func newTestServer() *server.Server {
	config := &server.Config{
		Address: ":8080",
		Debug:   true,
	}
	return server.NewServer(config)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.NotEmpty(t, response["time"])
}

func TestProcessXMLEndpoint(t *testing.T) {
	srv := newTestServer()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000001</InvoiceNo>
	<InvoiceSeries>KK23</InvoiceSeries>
	<Seller><TaxID>0123456789</TaxID><Name>Test Company</Name></Seller>
	<TotalAmount>1000000</TotalAmount>
</Invoice>`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/process/xml", bytes.NewReader([]byte(xmlData)))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response server.ProcessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "xml", response.Method)
	assert.Equal(t, 1.0, response.Confidence)
	require.NotNil(t, response.Invoice)
	assert.Equal(t, "0000001", response.Invoice.Number)
	assert.Equal(t, "KK23", response.Invoice.Series)
}

func TestProcessXMLEndpoint_EmptyBody(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/process/xml", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProcessXMLEndpoint_InvalidXML(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/process/xml", bytes.NewReader([]byte("not xml")))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestValidateEndpoint(t *testing.T) {
	srv := newTestServer()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000001</InvoiceNo>
	<Seller><TaxID>0123456789</TaxID></Seller>
	<TotalAmount>1000000</TotalAmount>
</Invoice>`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader([]byte(xmlData)))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response server.ValidationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Valid)
}

func TestInfoEndpoint(t *testing.T) {
	srv := newTestServer()

	xmlData := `<?xml version="1.0"?><Invoice/>`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/info", bytes.NewReader([]byte(xmlData)))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response server.InfoResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "xml", response.Format)
	assert.Greater(t, response.Size, 0)
}

func TestProcessAutoEndpoint_XML(t *testing.T) {
	srv := newTestServer()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000002</InvoiceNo>
	<Seller><TaxID>0123456789</TaxID></Seller>
</Invoice>`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/process/auto", bytes.NewReader([]byte(xmlData)))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response server.ProcessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "xml", response.Method)
	require.NotNil(t, response.Invoice)
	assert.Equal(t, "0000002", response.Invoice.Number)
}

func TestProcessImageEndpoint_NoLLM(t *testing.T) {
	srv := newTestServer() // No LLM configured

	// PNG magic bytes
	imageData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/process/image", bytes.NewReader(imageData))
	req.Header.Set("Content-Type", "image/png")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	// Should fail because no LLM is configured
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// Benchmark tests

func BenchmarkProcessXML(b *testing.B) {
	srv := newTestServer()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Invoice>
	<InvoiceNo>0000001</InvoiceNo>
	<InvoiceSeries>KK23</InvoiceSeries>
	<Seller><TaxID>0123456789</TaxID></Seller>
	<TotalAmount>1000000</TotalAmount>
</Invoice>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/process/xml", bytes.NewReader([]byte(xmlData)))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
	}
}

func BenchmarkHealth(b *testing.B) {
	srv := newTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
	}
}
