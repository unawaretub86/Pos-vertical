package service

import (
	"context"
	"testing"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// mixedTestEnv wires real repositories against an in-memory DB so mixed-payment
// behavior is verified against actual schema (migration12), not mocks.
type mixedTestEnv struct {
	svc       *SaleService
	sales     repository.SaleRepository
	customers repository.CustomerRepository
	registers repository.RegisterRepository
	products  repository.ProductRepository
	registerID int64
}

func newMixedTestEnv(t *testing.T) *mixedTestEnv {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	env := &mixedTestEnv{
		sales:     repository.NewSaleRepository(db.DB),
		customers: repository.NewCustomerRepository(db.DB),
		registers: repository.NewRegisterRepository(db.DB),
		products:  repository.NewProductRepository(db.DB),
	}
	env.svc = NewSaleService(env.sales, env.products, env.registers)

	reg, err := env.registers.Create(context.Background(), &model.Register{UserID: 1, OpeningAmount: 0})
	if err != nil {
		t.Fatalf("open register: %v", err)
	}
	env.registerID = reg.ID
	return env
}

func (e *mixedTestEnv) newProduct(t *testing.T, price float64) *model.Product {
	t.Helper()
	p, err := e.products.Create(context.Background(), &model.Product{
		Name: "Producto", Code: "P1", Type: "product", SalePrice: price, Unit: "unit", Stock: 9999,
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	return p
}

func (e *mixedTestEnv) newTaxedProduct(t *testing.T, price, taxRate float64) *model.Product {
	t.Helper()
	p, err := e.products.Create(context.Background(), &model.Product{
		Name: "Producto Gravado", Code: "P2", Type: "product", SalePrice: price, TaxRate: taxRate, Unit: "unit", Stock: 9999,
	})
	if err != nil {
		t.Fatalf("create taxed product: %v", err)
	}
	return p
}

// Doc test: venta de $60.000 con pagos [{cash,30000},{transfer,20000},{credit,10000}]
// → sales.payment_method='mixed', 3 filas en sale_payments, cliente queda con +10000 de saldo.
func TestSaleService_Create_MixedPayment(t *testing.T) {
	env := newMixedTestEnv(t)
	product := env.newProduct(t, 60000)
	customer, err := env.customers.Create(context.Background(), &model.Customer{Name: "Cliente Fiado"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	sale := &model.Sale{
		UserID:     1,
		CustomerID: &customer.ID,
		Total:      60000,
		Items:      []model.SaleItem{{ProductID: product.ID, Quantity: 1, UnitPrice: 60000}},
		Payments: []model.Payment{
			{Method: "cash", Amount: 30000},
			{Method: "transfer", Amount: 20000},
			{Method: "credit", Amount: 10000},
		},
	}

	result, err := env.svc.Create(context.Background(), sale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentMethod != "mixed" {
		t.Errorf("expected payment_method 'mixed', got %q", result.PaymentMethod)
	}
	if len(result.Payments) != 3 {
		t.Fatalf("expected 3 sale_payments rows, got %d", len(result.Payments))
	}

	balance, err := env.customers.Balance(context.Background(), customer.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if balance != 10000 {
		t.Errorf("expected customer balance 10000, got %.0f", balance)
	}
}

// Doc test: venta con un solo pago [{cash,15000}] sobre total $15.000 → se comporta
// igual que antes (payment_method='cash'), sin romper flujo existente.
func TestSaleService_Create_SinglePaymentBehavesLikeBefore(t *testing.T) {
	env := newMixedTestEnv(t)
	product := env.newProduct(t, 15000)

	sale := &model.Sale{
		UserID: 1,
		Total:  15000,
		Items:  []model.SaleItem{{ProductID: product.ID, Quantity: 1, UnitPrice: 15000}},
		Payments: []model.Payment{
			{Method: "cash", Amount: 15000},
		},
	}

	result, err := env.svc.Create(context.Background(), sale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PaymentMethod != "cash" {
		t.Errorf("expected payment_method 'cash', got %q", result.PaymentMethod)
	}
	if len(result.Payments) != 1 {
		t.Fatalf("expected 1 sale_payments row, got %d", len(result.Payments))
	}
}

// Doc test: pagos que suman $59.999 sobre total $60.000 (fuera de tolerancia) → rechazada con 400.
func TestSaleService_Create_PaymentsSumMismatch_Rejected(t *testing.T) {
	env := newMixedTestEnv(t)
	product := env.newProduct(t, 60000)

	sale := &model.Sale{
		UserID: 1,
		Total:  60000,
		Items:  []model.SaleItem{{ProductID: product.ID, Quantity: 1, UnitPrice: 60000}},
		Payments: []model.Payment{
			{Method: "cash", Amount: 59999},
		},
	}

	if _, err := env.svc.Create(context.Background(), sale); err == nil {
		t.Fatal("expected error: payments sum does not match total")
	}
}

// Regression: production incident — a mixed-payment sale of a taxed product
// ("nata", 5 x $10.000 @ 19% IVA) was rejected with "total inconsistente:
// esperado 59500, recibido 50000" because the caller (in production, the POS
// frontend) sent a payments[] sum and a `total` that both ignored tax. The
// earlier mixed-payment tests in this file all used untaxed products, so this
// combination — tax + mixed payments together — was never exercised until now.
func TestSaleService_Create_MixedPayment_WithTax(t *testing.T) {
	env := newMixedTestEnv(t)
	product := env.newTaxedProduct(t, 10000, 0.19)
	customer, err := env.customers.Create(context.Background(), &model.Customer{Name: "Cliente Fiado"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	// subtotal 5*10000=50000, tax 50000*0.19=9500, total 59500.
	sale := &model.Sale{
		UserID:     1,
		CustomerID: &customer.ID,
		Subtotal:   50000,
		Tax:        9500,
		Total:      59500,
		Items:      []model.SaleItem{{ProductID: product.ID, Quantity: 5, UnitPrice: 10000}},
		Payments: []model.Payment{
			{Method: "credit", Amount: 39500},
			{Method: "cash", Amount: 20000},
		},
	}

	result, err := env.svc.Create(context.Background(), sale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 59500 {
		t.Errorf("expected total 59500, got %.0f", result.Total)
	}
	if result.PaymentMethod != "mixed" {
		t.Errorf("expected payment_method 'mixed', got %q", result.PaymentMethod)
	}

	balance, err := env.customers.Balance(context.Background(), customer.ID)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	if balance != 39500 {
		t.Errorf("expected customer balance 39500, got %.0f", balance)
	}
}

// A payments[] sum that ignores tax (the exact shape of the production bug:
// client computed total from subtotal only, never adding tax) must be
// rejected, not silently accepted at the pre-tax total.
func TestSaleService_Create_MixedPayment_TaxOmittedFromPayments_Rejected(t *testing.T) {
	env := newMixedTestEnv(t)
	product := env.newTaxedProduct(t, 10000, 0.19)

	sale := &model.Sale{
		UserID: 1,
		Total:  50000, // wrong: should be 59500 once 19% tax is added
		Items:  []model.SaleItem{{ProductID: product.ID, Quantity: 5, UnitPrice: 10000}},
		Payments: []model.Payment{
			{Method: "cash", Amount: 30000},
			{Method: "transfer", Amount: 20000},
		},
	}

	if _, err := env.svc.Create(context.Background(), sale); err == nil {
		t.Fatal("expected error: total omits tax")
	}
}

// Doc test: cierre de caja con 1 venta mixta ($30k cash de $60k total) + 1 venta 100%
// cash ($20k) → expected_close cash = $50.000, no $80.000.
func TestRegisterRepo_SumCashSales_SplitsMixedPayments(t *testing.T) {
	env := newMixedTestEnv(t)
	product := env.newProduct(t, 60000)
	customer, err := env.customers.Create(context.Background(), &model.Customer{Name: "Cliente Fiado"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	mixedSale := &model.Sale{
		UserID:     1,
		CustomerID: &customer.ID,
		Total:      60000,
		Items:      []model.SaleItem{{ProductID: product.ID, Quantity: 1, UnitPrice: 60000}},
		Payments: []model.Payment{
			{Method: "cash", Amount: 30000},
			{Method: "transfer", Amount: 20000},
			{Method: "credit", Amount: 10000},
		},
	}
	if _, err := env.svc.Create(context.Background(), mixedSale); err != nil {
		t.Fatalf("create mixed sale: %v", err)
	}

	cashProduct := env.newProduct(t, 20000)
	cashSale := &model.Sale{
		UserID: 1,
		Total:  20000,
		Items:  []model.SaleItem{{ProductID: cashProduct.ID, Quantity: 1, UnitPrice: 20000}},
		Payments: []model.Payment{
			{Method: "cash", Amount: 20000},
		},
	}
	if _, err := env.svc.Create(context.Background(), cashSale); err != nil {
		t.Fatalf("create cash sale: %v", err)
	}

	sum, err := env.registers.SumCashSales(context.Background(), env.registerID)
	if err != nil {
		t.Fatalf("sum cash sales: %v", err)
	}
	if sum != 50000 {
		t.Errorf("expected cash sum 50000, got %.0f", sum)
	}
}
