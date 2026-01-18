package pdf_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/parser/pdf"
)

func TestNewExtractor(t *testing.T) {
	extractor := pdf.NewExtractor()
	require.NotNil(t, extractor)
}
