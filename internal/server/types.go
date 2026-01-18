package server

import (
	"github.com/rezonia/invoice-processor/internal/model"
)

// ProcessResponse is the response for process endpoints
type ProcessResponse struct {
	Invoice    *model.Invoice `json:"invoice"`
	Method     string         `json:"method"`
	Confidence float64        `json:"confidence"`
	Warnings   []string       `json:"warnings,omitempty"`
}

// ValidationResponse is the response for validate endpoint
type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// InfoResponse is the response for info endpoint
type InfoResponse struct {
	Format   string `json:"format"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error    string   `json:"error"`
	Details  string   `json:"details,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
