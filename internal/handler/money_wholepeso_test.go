package handler

// US-096 contract §4, T-6 (AC-11, Q-2) and US-102 (R-133) T-18 (AC-12): the
// whole-peso invariant over the full AC-1 domain (taxed and kg products).
// R-133 removed the moneyexact build tag, so it runs with go test ./... and
// rapid prints the shrunk counterexample if a fractional amount is stored.

import "testing"

func TestMoneyProperties_WholePeso(t *testing.T) {
	runMoneyMachine(t, moneyFullCatalog(), true)
}
