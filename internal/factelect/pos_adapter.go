package factelect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreyo/pos/internal/model"
	"github.com/andreyo/pos/internal/repository"
)

// PosSaleAdapter implements SaleAdapter for the POS vertical.
type PosSaleAdapter struct {
	salesRepo    repository.SaleRepository
	productsRepo repository.ProductRepository
	setupRepo    *repository.SetupRepository
	fiscalRepo   FiscalConfigRepository
}

func NewPosSaleAdapter(
	sales repository.SaleRepository,
	products repository.ProductRepository,
	setup *repository.SetupRepository,
	fiscal FiscalConfigRepository,
) *PosSaleAdapter {
	return &PosSaleAdapter{
		salesRepo:    sales,
		productsRepo: products,
		setupRepo:    setup,
		fiscalRepo:   fiscal,
	}
}

func (a *PosSaleAdapter) ToBillRequest(ctx context.Context, saleID int64, buyerEmail string) (*BillRequest, error) {
	sale, err := a.salesRepo.GetByID(ctx, saleID)
	if err != nil {
		return nil, fmt.Errorf("pos_adapter: obtener venta %d: %w", saleID, err)
	}

	cfg, err := a.fiscalRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("pos_adapter: obtener fiscal_config: %w", err)
	}

	setup, err := a.setupRepo.Get()
	if err != nil {
		return nil, fmt.Errorf("pos_adapter: obtener business_config: %w", err)
	}

	if cfg.FechaVencimiento != nil && time.Now().After(*cfg.FechaVencimiento) {
		return nil, errors.New("la resolución de facturación está vencida; renueva ante la DIAN")
	}

	issuer := IssuerInfo{
		NIT:         setup.NIT,
		RazonSocial: setup.Name,
		Direccion:   setup.Address,
		CIIU:        setup.CIIU,
		SoftwareID:  cfg.SoftwareID,
		SoftwarePIN: cfg.SoftwarePIN,
		Resolucion:  cfg.ResolucionDIAN,
		Prefijo:     cfg.Prefijo,
	}

	buyer := a.buildBuyer(sale, buyerEmail)

	items, totals, err := a.buildItems(ctx, sale)
	if err != nil {
		return nil, err
	}

	payments := []PaymentMethod{{
		Method: PaymentCode(sale.PaymentMethod),
		Amount: sale.Total,
	}}

	return &BillRequest{
		Issuer:         issuer,
		Buyer:          buyer,
		Items:          items,
		Totals:         totals,
		PaymentMethods: payments,
		IssuedAt:       sale.CreatedAt,
		Environment:    cfg.Ambiente,
	}, nil
}

func (a *PosSaleAdapter) buildBuyer(sale *model.Sale, buyerEmail string) BuyerInfo {
	// MVP: always anonymous. To enrich with customer fiscal data,
	// add CustomerRepository as a dependency and read document_type/document_number.
	return BuyerInfo{IsAnonymous: true, Email: buyerEmail}
}

func (a *PosSaleAdapter) buildItems(ctx context.Context, sale *model.Sale) ([]BillItem, BillTotals, error) {
	items := make([]BillItem, 0, len(sale.Items))
	var totalIVA float64
	for _, si := range sale.Items {
		prod, err := a.productsRepo.GetByID(ctx, si.ProductID)
		if err != nil {
			return nil, BillTotals{}, fmt.Errorf("pos_adapter: producto %d: %w", si.ProductID, err)
		}
		taxAmt := si.Subtotal * si.TaxRate
		totalIVA += taxAmt
		items = append(items, BillItem{
			Description:  prod.Name,
			ProductCode:  prod.Code,
			UnitCodeDIAN: UnitCode(prod.Unit),
			Quantity:     si.Quantity,
			UnitPrice:    si.UnitPrice,
			Discount:     si.Discount,
			TaxRate:      si.TaxRate * 100, // convert 0.19 → 19
			TaxAmount:    taxAmt,
			Subtotal:     si.Subtotal,
			ProductType:  productTypeCode(prod.Type),
		})
	}
	totals := BillTotals{
		Subtotal: sale.Subtotal,
		Discount: sale.Discount,
		TaxBase:  sale.Subtotal - sale.Discount,
		TotalIVA: totalIVA,
		Total:    sale.Total,
	}
	return items, totals, nil
}
