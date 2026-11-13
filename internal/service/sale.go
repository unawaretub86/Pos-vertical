package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

var validPaymentMethods = map[string]bool{
	"cash":     true,
	"transfer": true,
	"card":     true,
	"credit":   true,
}

// SaleService handles sale business logic.
type SaleService struct {
	repo         repository.SaleRepository
	productRepo  repository.ProductRepository
	registerRepo repository.RegisterRepository
	eledocRepo   repository.ElectronicDocRepository // always wired, also with FE disabled; Void's check is a no-op when the sale has no document
}

// WithElectronicDocs wires the electronic doc repository without changing the constructor.
func (s *SaleService) WithElectronicDocs(r repository.ElectronicDocRepository) {
	s.eledocRepo = r
}

// NewSaleService creates a SaleService backed by the given repositories.
func NewSaleService(repo repository.SaleRepository, productRepo repository.ProductRepository, registerRepo repository.RegisterRepository) *SaleService {
	return &SaleService{repo: repo, productRepo: productRepo, registerRepo: registerRepo}
}

func (s *SaleService) Create(ctx context.Context, sale *model.Sale) (*model.Sale, error) {
	// Legacy callers send a bare payment_method with no payments[]; validateSale
	// turns it into one line, which is re-priced to the server total below.
	legacy := len(sale.Payments) == 0 && sale.PaymentMethod != ""
	if err := validateSale(sale); err != nil {
		return nil, err
	}

	// Verify open register
	reg, err := s.registerRepo.GetOpen(ctx)
	if err != nil {
		return nil, errors.New("no hay caja abierta")
	}
	sale.RegisterID = &reg.ID

	// Price the sale server-side with the whole-peso rule (money.go): the
	// client subtotal and tax are never used.
	var subtotal, tax float64
	for i := range sale.Items {
		item := &sale.Items[i]
		lineSub := LineBase(item.Quantity, item.UnitPrice) - item.Discount
		item.Subtotal = lineSub
		subtotal += lineSub

		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("producto no encontrado (id=%d): %w", item.ProductID, err)
		}
		if product.Type == "product" && item.Quantity > product.Stock {
			return nil, fmt.Errorf("stock insuficiente para el producto '%s' (disponible: %.0f, solicitado: %.0f)", product.Name, product.Stock, item.Quantity)
		}
		item.TaxRate = product.TaxRate
		tax += LineTax(lineSub, product.TaxRate)

		// Detect price override
		if math.Abs(item.UnitPrice-product.SalePrice) > 0.01 {
			claims, _ := ctx.Value(appmw.ClaimsKey).(*model.Claims)
			if claims == nil || claims.Role != "admin" {
				return nil, fmt.Errorf("solo un administrador puede modificar el precio del producto '%s'", product.Name)
			}
			item.OriginalPrice = product.SalePrice
		}
	}
	if sale.Discount > subtotal {
		return nil, errors.New("el descuento no puede superar el subtotal de la venta")
	}
	sale.Subtotal = subtotal
	sale.Tax = tax
	recalcTotal := subtotal - sale.Discount + tax
	if !sale.NoClientTotal && sale.Total != recalcTotal {
		return nil, fmt.Errorf("total inconsistente: esperado %.0f, recibido %.0f", recalcTotal, sale.Total)
	}
	if legacy {
		if recalcTotal > 0 {
			sale.Payments[0].Amount = recalcTotal
		} else {
			sale.Payments = nil
		}
	}
	if len(sale.Payments) == 0 && recalcTotal > 0 {
		return nil, errors.New("la venta debe tener al menos un método de pago")
	}
	if paymentsSum := sumPayments(sale.Payments); paymentsSum != recalcTotal {
		return nil, fmt.Errorf("la suma de los pagos ($%.0f) no coincide con el total de la venta ($%.0f)", paymentsSum, recalcTotal)
	}
	sale.Total = recalcTotal

	switch {
	case len(sale.Payments) > 1:
		sale.PaymentMethod = "mixed"
	case len(sale.Payments) == 1:
		sale.PaymentMethod = sale.Payments[0].Method
	default:
		// Zero-total sale (every line discounted 100%): no payment rows.
		sale.PaymentMethod = "cash"
	}

	if sale.Status == "" {
		sale.Status = "completed"
	}
	return s.repo.Create(ctx, sale)
}

func (s *SaleService) List(ctx context.Context, filter repository.SaleFilter) ([]*model.Sale, error) {
	return s.repo.List(ctx, filter)
}

func (s *SaleService) GetByID(ctx context.Context, id int64) (*model.Sale, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SaleService) Void(ctx context.Context, saleID int64, userID int64) (*model.Sale, error) {
	if s.eledocRepo != nil {
		doc, err := s.eledocRepo.GetBySaleID(ctx, saleID)
		if err == nil && doc != nil &&
			(doc.Status == "accepted" || doc.Status == "pending") {
			return nil, &repository.ErrRequiresCreditNote{
				DocumentID: doc.ID,
				CUDE:       doc.CUDE,
			}
		}
	}
	return s.repo.Void(ctx, saleID, userID)
}

// validateSale runs the checks that need no product data: whole-peso inputs,
// discount bounds and the shape of the payment lines. Pricing-dependent checks
// (client total, payments sum) are in Create.
func validateSale(sale *model.Sale) error {
	if len(sale.Payments) == 0 && sale.PaymentMethod != "" {
		amount := sale.Total
		if amount <= 0 {
			amount = 1 // placeholder, replaced by the server total in Create
		}
		sale.Payments = []model.Payment{{Method: sale.PaymentMethod, Amount: amount}}
	}
	if len(sale.Items) == 0 {
		return errors.New("la venta debe tener al menos un artículo")
	}
	if !IsWholePeso(sale.Discount) {
		return errors.New("el descuento debe ser un número entero de pesos")
	}
	if sale.Discount < 0 {
		return errors.New("el descuento no puede ser negativo")
	}
	if !sale.NoClientTotal && !IsWholePeso(sale.Total) {
		return errors.New("el total debe ser un número entero de pesos")
	}
	if sale.CashTendered != nil && !IsWholePeso(*sale.CashTendered) {
		return errors.New("el efectivo recibido debe ser un número entero de pesos")
	}
	if sale.ChangeAmount != nil && !IsWholePeso(*sale.ChangeAmount) {
		return errors.New("el cambio debe ser un número entero de pesos")
	}
	for _, item := range sale.Items {
		if item.Quantity <= 0 {
			return errors.New("la cantidad de cada artículo debe ser mayor a cero")
		}
		if item.UnitPrice < 0 {
			return errors.New("el precio unitario no puede ser negativo")
		}
		if !IsWholePeso(item.UnitPrice) {
			return errors.New("el precio unitario debe ser un número entero de pesos")
		}
		if !IsWholePeso(item.Discount) {
			return errors.New("el descuento por artículo debe ser un número entero de pesos")
		}
		if item.Discount < 0 {
			return errors.New("el descuento por artículo no puede ser negativo")
		}
		if item.Discount > LineBase(item.Quantity, item.UnitPrice) {
			return errors.New("el descuento por artículo no puede superar el subtotal del artículo")
		}
	}
	if len(sale.Payments) > 3 {
		return errors.New("no se pueden combinar más de 3 métodos de pago")
	}
	requiresCustomer := false
	for _, p := range sale.Payments {
		if !validPaymentMethods[p.Method] {
			return errors.New("método de pago inválido: debe ser cash, transfer, card o credit")
		}
		if !IsWholePeso(p.Amount) {
			return errors.New("el monto de cada método de pago debe ser un número entero de pesos")
		}
		if p.Amount <= 0 {
			return errors.New("el monto de cada método de pago debe ser mayor a cero")
		}
		if p.Method == "credit" {
			requiresCustomer = true
		}
	}
	if requiresCustomer && sale.CustomerID == nil {
		return errors.New("el método de pago 'credito' requiere un cliente")
	}
	return nil
}

func sumPayments(payments []model.Payment) float64 {
	var sum float64
	for _, p := range payments {
		sum += p.Amount
	}
	return sum
}
