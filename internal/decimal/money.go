package decimal

import (
	"github.com/shopspring/decimal"
)

// Zero is decimal zero
var Zero = decimal.Zero

// FromInt creates decimal from int (common for VND)
func FromInt(v int64) decimal.Decimal {
	return decimal.NewFromInt(v)
}

// FromFloat creates decimal from float with rounding
func FromFloat(v float64) decimal.Decimal {
	return decimal.NewFromFloat(v).Round(2)
}

// FromString parses decimal from string
func FromString(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// MustFromString parses decimal from string, panics on error
func MustFromString(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Mul multiplies two decimals, rounds to 2 places
func Mul(a, b decimal.Decimal) decimal.Decimal {
	return a.Mul(b).Round(2)
}

// Div divides a by b, rounds to 2 places
func Div(a, b decimal.Decimal) decimal.Decimal {
	if b.IsZero() {
		return Zero
	}
	return a.Div(b).Round(2)
}

// CalculateVAT computes VAT amount: amount * (rate/100)
// Rounds to 0 decimals (VND has no cents)
func CalculateVAT(amount decimal.Decimal, ratePercent int) decimal.Decimal {
	if ratePercent == 0 {
		return Zero
	}
	rate := decimal.NewFromInt(int64(ratePercent))
	hundred := decimal.NewFromInt(100)
	return amount.Mul(rate).Div(hundred).Round(0)
}

// CalculateLineTotal computes: amount - discount + vat
func CalculateLineTotal(amount, discount, vat decimal.Decimal) decimal.Decimal {
	return amount.Sub(discount).Add(vat).Round(0)
}

// CalculatePercentage computes: amount * (percentage/100)
func CalculatePercentage(amount decimal.Decimal, percentage decimal.Decimal) decimal.Decimal {
	hundred := decimal.NewFromInt(100)
	return amount.Mul(percentage).Div(hundred).Round(0)
}

// Sum sums a slice of decimals
func Sum(values []decimal.Decimal) decimal.Decimal {
	result := Zero
	for _, v := range values {
		result = result.Add(v)
	}
	return result
}

// IsPositive returns true if decimal is greater than zero
func IsPositive(d decimal.Decimal) bool {
	return d.GreaterThan(Zero)
}

// IsNonNegative returns true if decimal is >= zero
func IsNonNegative(d decimal.Decimal) bool {
	return d.GreaterThanOrEqual(Zero)
}

// RoundVND rounds to whole number (VND has no decimals)
func RoundVND(d decimal.Decimal) decimal.Decimal {
	return d.Round(0)
}
