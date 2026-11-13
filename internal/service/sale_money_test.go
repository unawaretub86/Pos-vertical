package service

// US-102 (R-133) contract §5, T-1 .. T-5 and T-22: the whole-peso rounding
// rule of §3 at the service level. Half away from zero, per line:
// lineBase = round(qty x unit), lineTax = round(lineSubtotal x rate).

import (
	"context"
	"strings"
	"testing"

	"github.com/andreyo/pos/internal/model"
)

// moneySvc builds a SaleService whose single product costs price with the
// given IVA rate (the unit price of every line equals it, so no override).
func moneySvc(price, rate float64) (*SaleService, *mockSaleRepo) {
	repo := &mockSaleRepo{}
	svc := NewSaleService(repo,
		&mockProductRepo{salePrice: price, taxRate: rate},
		&mockRegisterRepo{openRegister: openRegister()})
	return svc, repo
}

type moneyWant struct {
	subtotal, tax, total float64
}

func createCash(t *testing.T, svc *SaleService, items []model.SaleItem, discount, clientTotal float64) (*model.Sale, error) {
	t.Helper()
	return svc.Create(context.Background(), &model.Sale{
		Discount: discount,
		Total:    clientTotal,
		Items:    items,
		Payments: []model.Payment{{Method: "cash", Amount: clientTotal}},
	})
}

func assertSale(t *testing.T, what string, repo *mockSaleRepo, want moneyWant) {
	t.Helper()
	got := repo.created
	if got == nil {
		t.Fatalf("%s: no sale was stored", what)
	}
	if got.Subtotal != want.subtotal || got.Tax != want.tax || got.Total != want.total {
		t.Errorf("%s: stored subtotal/tax/total = %v/%v/%v, want %v/%v/%v",
			what, got.Subtotal, got.Tax, got.Total, want.subtotal, want.tax, want.total)
	}
}

// T-1: $10.001 x 1 at 19% -> 10001 / 1900 / 11901 (was 11901.19).
func TestSaleMoney_T1_TaxRoundedAndClientTotalExact(t *testing.T) {
	svc, repo := moneySvc(10001, 0.19)
	items := []model.SaleItem{{ProductID: 1, Quantity: 1, UnitPrice: 10001}}
	if _, err := createCash(t, svc, items, 0, 11901); err != nil {
		t.Fatalf("client total 11901: %v", err)
	}
	assertSale(t, "P1 x 1", repo, moneyWant{10001, 1900, 11901})
	if got := repo.created.Items[0].Subtotal; got != 10001 {
		t.Errorf("line subtotal = %v, want 10001", got)
	}

	for _, bad := range []float64{11902, 11900} {
		svc, repo := moneySvc(10001, 0.19)
		_, err := createCash(t, svc, items, 0, bad)
		want := "total inconsistente: esperado 11901, recibido " + map[float64]string{11902: "11902", 11900: "11900"}[bad]
		if err == nil || err.Error() != want {
			t.Errorf("client total %v: error %v, want %q", bad, err, want)
		}
		if repo.created != nil {
			t.Errorf("client total %v: a sale was stored", bad)
		}
	}
}

// T-2: 3 lines of $1.001 at 19% -> 190 x 3 = 570, total 3573 (per line; per
// sale would give 571 and 3574).
func TestSaleMoney_T2_TaxIsRoundedPerLine(t *testing.T) {
	svc, repo := moneySvc(1001, 0.19)
	items := []model.SaleItem{
		{ProductID: 1, Quantity: 1, UnitPrice: 1001},
		{ProductID: 1, Quantity: 1, UnitPrice: 1001},
		{ProductID: 1, Quantity: 1, UnitPrice: 1001},
	}
	if _, err := createCash(t, svc, items, 0, 3573); err != nil {
		t.Fatalf("3 x Cafe, client total 3573: %v", err)
	}
	assertSale(t, "3 x P2", repo, moneyWant{3003, 570, 3573})

	svc2, repo2 := moneySvc(1001, 0.19)
	_, err := createCash(t, svc2, items, 0, 3574)
	if err == nil || err.Error() != "total inconsistente: esperado 3573, recibido 3574" {
		t.Errorf("client total 3574: error %v, want total inconsistente 3573/3574", err)
	}
	if repo2.created != nil {
		t.Errorf("client total 3574: a sale was stored")
	}
}

// T-3: half away from zero. $50 at 5% -> 2.5 -> 3; $1.050 at 19% -> 199.5 -> 200.
func TestSaleMoney_T3_HalfRoundsUp(t *testing.T) {
	cases := []struct {
		name        string
		price, rate float64
		want        moneyWant
	}{
		{"Panela $50 at 5%", 50, 0.05, moneyWant{50, 3, 53}},
		{"$1.050 at 19%", 1050, 0.19, moneyWant{1050, 200, 1250}},
	}
	for _, c := range cases {
		svc, repo := moneySvc(c.price, c.rate)
		items := []model.SaleItem{{ProductID: 1, Quantity: 1, UnitPrice: c.price}}
		if _, err := createCash(t, svc, items, 0, c.want.total); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		assertSale(t, c.name, repo, c.want)
	}
}

// T-4: quantity fractions of a kg product at $9.999, tax 0.
func TestSaleMoney_T4_QuantityFractions(t *testing.T) {
	cases := []struct {
		qty        float64
		base       float64
		unroundedV string
	}{
		{0.5, 5000, "4999.5"},
		{0.1, 1000, "999.9"},
		{0.125, 1250, "1249.875"},
	}
	for _, c := range cases {
		svc, repo := moneySvc(9999, 0)
		items := []model.SaleItem{{ProductID: 1, Quantity: c.qty, UnitPrice: 9999}}
		if _, err := createCash(t, svc, items, 0, c.base); err != nil {
			t.Fatalf("%v kg (unrounded %s): %v", c.qty, c.unroundedV, err)
		}
		assertSale(t, "kg line", repo, moneyWant{c.base, 0, c.base})
		if got := repo.created.Items[0].Subtotal; got != c.base {
			t.Errorf("%v kg: line subtotal = %v, want %v", c.qty, got, c.base)
		}
	}
}

// T-5: item discount lowers the tax base; the global discount does not.
func TestSaleMoney_T5_Discounts(t *testing.T) {
	svc, repo := moneySvc(10001, 0.19)
	items := []model.SaleItem{{ProductID: 1, Quantity: 1, UnitPrice: 10001, Discount: 1}}
	// lineSubtotal 10000, tax round(1900) = 1900, total 11900.
	if _, err := createCash(t, svc, items, 0, 11900); err != nil {
		t.Fatalf("item discount $1: %v", err)
	}
	assertSale(t, "item discount 1", repo, moneyWant{10000, 1900, 11900})

	svc2, repo2 := moneySvc(10001, 0.19)
	items2 := []model.SaleItem{{ProductID: 1, Quantity: 1, UnitPrice: 10001}}
	// subtotal 10001, tax 1900 (base not reduced), total 10001 - 500 + 1900 = 11401.
	if _, err := createCash(t, svc2, items2, 500, 11401); err != nil {
		t.Fatalf("global discount $500: %v", err)
	}
	assertSale(t, "global discount 500", repo2, moneyWant{10001, 1900, 11401})
}

// T-22: rates outside 0/5/19 go through the same rule (basis points).
func TestSaleMoney_T22_OtherRates(t *testing.T) {
	cases := []struct {
		name        string
		price, rate float64
		want        moneyWant
	}{
		{"Gaseosa $1.001 at 8%", 1001, 0.08, moneyWant{1001, 80, 1081}},
		{"$1.050 at 8%", 1050, 0.08, moneyWant{1050, 84, 1134}},
		{"$1.001 at 2,5%", 1001, 0.025, moneyWant{1001, 25, 1026}},
	}
	for _, c := range cases {
		svc, repo := moneySvc(c.price, c.rate)
		items := []model.SaleItem{{ProductID: 1, Quantity: 1, UnitPrice: c.price}}
		if _, err := createCash(t, svc, items, 0, c.want.total); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		assertSale(t, c.name, repo, c.want)
	}
}

// The error text of a rejected total must name both figures (AC-2).
func TestSaleMoney_T1_ErrorNamesBothTotals(t *testing.T) {
	svc, _ := moneySvc(10001, 0.19)
	items := []model.SaleItem{{ProductID: 1, Quantity: 1, UnitPrice: 10001}}
	_, err := createCash(t, svc, items, 0, 11902)
	if err == nil || !strings.Contains(err.Error(), "esperado 11901") {
		t.Errorf("error = %v, want it to contain 'esperado 11901'", err)
	}
}
