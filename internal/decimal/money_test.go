package decimal_test

import (
	"testing"

	dec "github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rezonia/invoice-processor/internal/decimal"
)

func TestFromInt(t *testing.T) {
	d := decimal.FromInt(100000)
	assert.True(t, d.Equal(dec.NewFromInt(100000)))
}

func TestFromFloat(t *testing.T) {
	d := decimal.FromFloat(100.555)
	// Should round to 2 decimal places
	assert.True(t, d.Equal(dec.NewFromFloat(100.56)))
}

func TestFromString(t *testing.T) {
	d, err := decimal.FromString("123456.78")
	require.NoError(t, err)
	assert.True(t, d.Equal(dec.RequireFromString("123456.78")))

	_, err = decimal.FromString("not-a-number")
	require.Error(t, err)
}

func TestMustFromString(t *testing.T) {
	d := decimal.MustFromString("999.99")
	assert.True(t, d.Equal(dec.RequireFromString("999.99")))

	assert.Panics(t, func() {
		decimal.MustFromString("invalid")
	})
}

func TestMul(t *testing.T) {
	a := dec.NewFromInt(100)
	b := dec.NewFromFloat(0.15)
	result := decimal.Mul(a, b)
	assert.True(t, result.Equal(dec.NewFromInt(15)))
}

func TestDiv(t *testing.T) {
	a := dec.NewFromInt(100)
	b := dec.NewFromInt(3)
	result := decimal.Div(a, b)
	assert.True(t, result.Equal(dec.RequireFromString("33.33")))

	// Division by zero returns zero
	result = decimal.Div(a, dec.Zero)
	assert.True(t, result.IsZero())
}

func TestCalculateVAT(t *testing.T) {
	tests := []struct {
		name        string
		amount      int64
		ratePercent int
		expected    int64
	}{
		{"10% of 1000000", 1000000, 10, 100000},
		{"5% of 1000000", 1000000, 5, 50000},
		{"0% of 1000000", 1000000, 0, 0},
		{"10% of 999999 (rounds to nearest)", 999999, 10, 100000},
		{"10% of 555555", 555555, 10, 55556}, // rounds to nearest
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := dec.NewFromInt(tt.amount)
			result := decimal.CalculateVAT(amount, tt.ratePercent)
			expected := dec.NewFromInt(tt.expected)

			assert.True(t, result.Equal(expected),
				"amount=%d, rate=%d%%: got %s, want %d",
				tt.amount, tt.ratePercent, result.String(), tt.expected)
		})
	}
}

func TestCalculateLineTotal(t *testing.T) {
	amount := dec.NewFromInt(1000000)
	discount := dec.NewFromInt(100000)
	vat := dec.NewFromInt(90000)

	// Total = 1000000 - 100000 + 90000 = 990000
	result := decimal.CalculateLineTotal(amount, discount, vat)
	assert.True(t, result.Equal(dec.NewFromInt(990000)))
}

func TestCalculatePercentage(t *testing.T) {
	amount := dec.NewFromInt(500000)
	percentage := dec.NewFromInt(15)

	// 15% of 500000 = 75000
	result := decimal.CalculatePercentage(amount, percentage)
	assert.True(t, result.Equal(dec.NewFromInt(75000)))
}

func TestSum(t *testing.T) {
	values := []dec.Decimal{
		dec.NewFromInt(100),
		dec.NewFromInt(200),
		dec.NewFromInt(300),
	}
	result := decimal.Sum(values)
	assert.True(t, result.Equal(dec.NewFromInt(600)))
}

func TestSum_Empty(t *testing.T) {
	result := decimal.Sum([]dec.Decimal{})
	assert.True(t, result.IsZero())
}

func TestIsPositive(t *testing.T) {
	assert.True(t, decimal.IsPositive(dec.NewFromInt(1)))
	assert.False(t, decimal.IsPositive(dec.Zero))
	assert.False(t, decimal.IsPositive(dec.NewFromInt(-1)))
}

func TestIsNonNegative(t *testing.T) {
	assert.True(t, decimal.IsNonNegative(dec.NewFromInt(1)))
	assert.True(t, decimal.IsNonNegative(dec.Zero))
	assert.False(t, decimal.IsNonNegative(dec.NewFromInt(-1)))
}

func TestRoundVND(t *testing.T) {
	// VND has no decimals
	d := dec.RequireFromString("123456.789")
	result := decimal.RoundVND(d)
	assert.True(t, result.Equal(dec.NewFromInt(123457))) // rounds up
}
