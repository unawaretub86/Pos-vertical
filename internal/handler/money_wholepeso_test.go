//go:build moneyexact

package handler

// US-096 contract §4, T-6 (AC-11, Q-2): the whole-peso invariant over the
// full AC-1 domain (taxed and kg products). It is red on purpose until R-133
// moves money to whole pesos: run it with
//
//	go test -tags moneyexact -run TestMoneyProperties_WholePeso ./internal/handler/
//
// and rapid prints the shrunk counterexample (for example 1 x Gaseosa $1.001
// at 19% -> sales.tax 190.19). R-133 removes the build tag.

import "testing"

func TestMoneyProperties_WholePeso(t *testing.T) {
	runMoneyMachine(t, moneyFullCatalog(), true)
}
