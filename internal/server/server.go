package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rezonia/invoice-processor/internal/llm"
	"github.com/rezonia/invoice-processor/internal/processor"
)

// Config holds server configuration
type Config struct {
	Address        string
	APIKey         string
	LLMBaseURL     string
	LLMModel       string
	LLMVisionModel string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	Debug          bool
}

// Server represents the HTTP API server
type Server struct {
	config   *Config
	router   *gin.Engine
	pipeline *processor.Pipeline
}

// NewServer creates a new API server
func NewServer(config *Config) *Server {
	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	if config.Debug {
		router.Use(gin.Logger())
	}

	// Create LLM extractor if API key provided
	var llmExtractor *llm.Extractor
	if config.APIKey != "" {
		// Build client options
		var clientOpts []llm.ClientOption
		if config.LLMBaseURL != "" {
			clientOpts = append(clientOpts, llm.WithBaseURL(config.LLMBaseURL))
		}

		client := llm.NewClient(config.APIKey, clientOpts...)

		// Build extractor options
		var extractorOpts []llm.ExtractorOption
		if config.LLMModel != "" {
			extractorOpts = append(extractorOpts, llm.WithTextModel(config.LLMModel))
		}
		if config.LLMVisionModel != "" {
			extractorOpts = append(extractorOpts, llm.WithVisionModel(config.LLMVisionModel))
		}

		llmExtractor = llm.NewExtractor(client, extractorOpts...)
	}

	// Create pipeline
	pipeline := processor.NewPipeline(
		processor.WithLLMExtractor(llmExtractor),
	)

	s := &Server{
		config:   config,
		router:   router,
		pipeline: pipeline,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handleHealth)

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// Process endpoints
		v1.POST("/process/xml", s.handleProcessXML)
		v1.POST("/process/pdf", s.handleProcessPDF)
		v1.POST("/process/image", s.handleProcessImage)
		v1.POST("/process/auto", s.handleProcessAuto)

		// Validate endpoint
		v1.POST("/validate", s.handleValidate)

		// Info endpoint
		v1.POST("/info", s.handleInfo)
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         s.config.Address,
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}
	return srv.ListenAndServe()
}

// Handler returns the http.Handler for use with custom servers
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleProcessXML(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result := s.pipeline.ProcessXMLBytes(ctx, body)
	if result.Error != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":    result.Error.Error(),
			"warnings": result.Warnings,
		})
		return
	}

	c.JSON(http.StatusOK, ProcessResponse{
		Invoice:    result.Invoice,
		Method:     string(result.Method),
		Confidence: result.Confidence,
		Warnings:   result.Warnings,
	})
}

func (s *Server) handleProcessPDF(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	// Get image data if provided as query param (for LLM vision fallback)
	var imageData []byte
	var mimeType string

	result := s.pipeline.ProcessPDF(ctx, nil, imageData, mimeType)
	if result.Error != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":    result.Error.Error(),
			"warnings": result.Warnings,
		})
		return
	}

	c.JSON(http.StatusOK, ProcessResponse{
		Invoice:    result.Invoice,
		Method:     string(result.Method),
		Confidence: result.Confidence,
		Warnings:   result.Warnings,
	})
}

func (s *Server) handleProcessImage(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	// Get content type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	result := s.pipeline.ProcessImage(ctx, body, contentType)
	if result.Error != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":    result.Error.Error(),
			"warnings": result.Warnings,
		})
		return
	}

	c.JSON(http.StatusOK, ProcessResponse{
		Invoice:    result.Invoice,
		Method:     string(result.Method),
		Confidence: result.Confidence,
		Warnings:   result.Warnings,
	})
}

func (s *Server) handleProcessAuto(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	// Auto-detect format
	format := processor.DetectFormat(body)
	contentType := c.GetHeader("Content-Type")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	var result *processor.Result

	switch format {
	case processor.FormatXML:
		result = s.pipeline.ProcessXMLBytes(ctx, body)

	case processor.FormatPDF:
		result = s.pipeline.ProcessPDF(ctx, nil, body, "application/pdf")

	case processor.FormatImage:
		mimeType := contentType
		if mimeType == "" || mimeType == "application/octet-stream" {
			mimeType = detectMimeType(body)
		}
		result = s.pipeline.ProcessImage(ctx, body, mimeType)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file format"})
		return
	}

	if result.Error != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":    result.Error.Error(),
			"warnings": result.Warnings,
		})
		return
	}

	c.JSON(http.StatusOK, ProcessResponse{
		Invoice:    result.Invoice,
		Method:     string(result.Method),
		Confidence: result.Confidence,
		Warnings:   result.Warnings,
	})
}

func (s *Server) handleValidate(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	// Only validate XML files
	format := processor.DetectFormat(body)
	if format != processor.FormatXML {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only XML validation is supported"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result := s.pipeline.ProcessXMLBytes(ctx, body)
	if result.Error != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"valid":  false,
			"errors": []string{result.Error.Error()},
		})
		return
	}

	// Validate the invoice
	errors, warnings := validateInvoice(result)
	valid := len(errors) == 0

	c.JSON(http.StatusOK, ValidationResponse{
		Valid:    valid,
		Errors:   errors,
		Warnings: warnings,
	})
}

func (s *Server) handleInfo(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	format := processor.DetectFormat(body)
	mimeType := detectMimeType(body)

	c.JSON(http.StatusOK, InfoResponse{
		Format:   format.String(),
		MimeType: mimeType,
		Size:     len(body),
	})
}

// Helper functions

func detectMimeType(data []byte) string {
	if len(data) < 8 {
		return "application/octet-stream"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	// TIFF
	if (data[0] == 0x49 && data[1] == 0x49) || (data[0] == 0x4D && data[1] == 0x4D) {
		return "image/tiff"
	}
	// PDF
	if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
		return "application/pdf"
	}
	// XML
	if data[0] == '<' || (data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF && data[3] == '<') {
		return "application/xml"
	}

	return "application/octet-stream"
}

func validateInvoice(inv *processor.Result) ([]string, []string) {
	var errors, warnings []string

	if inv == nil || inv.Invoice == nil {
		return []string{"no invoice data"}, nil
	}

	invoice := inv.Invoice

	// Required fields
	if invoice.Number == "" {
		errors = append(errors, "missing invoice number")
	}
	if invoice.Date.IsZero() {
		warnings = append(warnings, "missing invoice date")
	}
	if invoice.Seller.TaxID == "" {
		errors = append(errors, "missing seller tax ID")
	}

	// Amount validation
	if invoice.TotalAmount.IsZero() {
		warnings = append(warnings, "total amount is zero or missing")
	}

	// Check calculation
	if !invoice.SubtotalAmount.IsZero() && !invoice.TaxAmount.IsZero() && !invoice.TotalAmount.IsZero() {
		expected := invoice.SubtotalAmount.Add(invoice.TaxAmount)
		if !expected.Equal(invoice.TotalAmount) {
			warnings = append(warnings, "amount calculation mismatch")
		}
	}

	return errors, warnings
}
