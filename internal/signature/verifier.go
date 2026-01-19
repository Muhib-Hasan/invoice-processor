package signature

import "context"

// Format constants for document types
const (
	FormatXML = "xml"
	FormatPDF = "pdf"
)

// Verifier defines the interface for signature verification
type Verifier interface {
	// Verify verifies the digital signature on the given data
	// Returns VerificationResult with detailed check outcomes
	Verify(ctx context.Context, data []byte) (*VerificationResult, error)

	// CanVerify returns true if this verifier can handle the given data format
	CanVerify(data []byte) bool

	// Format returns the format this verifier handles (xml, pdf)
	Format() string
}

// VerifierRegistry holds registered verifiers for different formats
type VerifierRegistry struct {
	verifiers []Verifier
}

// NewVerifierRegistry creates a new empty registry
func NewVerifierRegistry() *VerifierRegistry {
	return &VerifierRegistry{
		verifiers: make([]Verifier, 0),
	}
}

// Register adds a verifier to the registry
func (r *VerifierRegistry) Register(v Verifier) {
	r.verifiers = append(r.verifiers, v)
}

// Detect finds a verifier that can handle the given data
func (r *VerifierRegistry) Detect(data []byte) (Verifier, error) {
	for _, v := range r.verifiers {
		if v.CanVerify(data) {
			return v, nil
		}
	}
	return nil, ErrUnsupportedFormat("unknown")
}

// Verify verifies signature using the appropriate verifier
func (r *VerifierRegistry) Verify(ctx context.Context, data []byte) (*VerificationResult, error) {
	verifier, err := r.Detect(data)
	if err != nil {
		return nil, err
	}
	return verifier.Verify(ctx, data)
}

// GetVerifier returns a verifier for a specific format
func (r *VerifierRegistry) GetVerifier(format string) Verifier {
	for _, v := range r.verifiers {
		if v.Format() == format {
			return v
		}
	}
	return nil
}

// AvailableFormats returns list of formats that can be verified
func (r *VerifierRegistry) AvailableFormats() []string {
	formats := make([]string, 0, len(r.verifiers))
	for _, v := range r.verifiers {
		formats = append(formats, v.Format())
	}
	return formats
}
