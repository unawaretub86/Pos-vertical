package service

import (
	"errors"
	"math"
)

// ErrNotWholePeso is returned when a money input has a fraction of a peso.
var ErrNotWholePeso = errors.New("el monto debe ser un número entero de pesos")

// Whole-peso money rule (US-102, contract §3). Every peso amount is a whole
// number; fractions come only from quantity x price and from tax rates, and
// are rounded half away from zero with integer arithmetic so that Go and the
// frontend (frontend/lib/saleCalc.ts) agree on every half.

// IsWholePeso reports whether v is a whole number of pesos.
func IsWholePeso(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v == math.Trunc(v)
}

// divRound returns num/den rounded half away from zero (den > 0).
func divRound(num, den int64) int64 {
	if num < 0 {
		return -((-num + den/2) / den)
	}
	return (num + den/2) / den
}

// RoundHalfUp rounds v half away from zero to a whole number.
func RoundHalfUp(v float64) float64 {
	return math.Round(v)
}

// LineBase returns round(quantity x unitPrice), with the quantity taken in
// thousandths (the catalog allows at most 3 decimals).
func LineBase(quantity, unitPrice float64) float64 {
	milli := int64(math.Round(quantity * 1000))
	return float64(divRound(milli*int64(math.Round(unitPrice)), 1000))
}

// LineTax returns round(lineSubtotal x rate) using basis points of the rate
// (19% -> 1900).
func LineTax(lineSubtotal, taxRate float64) float64 {
	bp := int64(math.Round(taxRate * 10000))
	return float64(divRound(int64(math.Round(lineSubtotal))*bp, 10000))
}
