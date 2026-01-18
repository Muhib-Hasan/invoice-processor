package xml

import (
	"bytes"
	"context"
	"io"

	"github.com/rezonia/invoice-processor/internal/model"
)

// Adapter parses provider-specific XML into Invoice
type Adapter interface {
	// Parse parses XML content into Invoice
	Parse(ctx context.Context, r io.Reader) (*model.Invoice, error)

	// CanParse returns true if adapter can handle this content
	CanParse(content []byte) bool

	// Provider returns the provider type
	Provider() model.Provider
}

// Registry holds all registered adapters
type Registry struct {
	adapters []Adapter
}

// NewRegistry creates registry with all adapters
// Order matters: more specific adapters should come before generic ones
func NewRegistry() *Registry {
	return &Registry{
		adapters: []Adapter{
			NewVNPTAdapter(),    // <SInvoice> - unique
			NewViettelAdapter(), // <HDon> - unique, must be before MISA (both use <MST>)
			NewFPTAdapter(),     // <EInvoice> - unique
			NewMISAAdapter(),    // <MST>, Vietnamese fields
			NewTCTAdapter(),     // Generic <Invoice><TaxID> - most generic, last
		},
	}
}

// Detect identifies provider from XML content
func (r *Registry) Detect(content []byte) (Adapter, error) {
	for _, a := range r.adapters {
		if a.CanParse(content) {
			return a, nil
		}
	}
	return nil, model.NewParseError(model.ProviderUnknown, "root", "unknown XML format, no matching adapter found", nil)
}

// Parse parses XML using appropriate adapter
func (r *Registry) Parse(ctx context.Context, content []byte) (*model.Invoice, error) {
	adapter, err := r.Detect(content)
	if err != nil {
		return nil, err
	}
	return adapter.Parse(ctx, bytes.NewReader(content))
}

// RegisterAdapter adds a custom adapter to the registry
func (r *Registry) RegisterAdapter(a Adapter) {
	// Add at the beginning so custom adapters take priority
	r.adapters = append([]Adapter{a}, r.adapters...)
}

// GetAdapter returns adapter for a specific provider
func (r *Registry) GetAdapter(provider model.Provider) Adapter {
	for _, a := range r.adapters {
		if a.Provider() == provider {
			return a
		}
	}
	return nil
}
