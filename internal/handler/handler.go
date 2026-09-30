package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/factelect"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Handler holds all HTTP handler methods and their dependencies.
type Handler struct {
	db           *database.DB
	categories   *service.CategoryService
	products     *service.ProductService
	sales        *service.SaleService
	registers    *service.RegisterService
	customers    *service.CustomerService
	users        *service.UserService
	setup        *repository.SetupRepository
	stockEntries repository.StockEntryRepository
	reports      repository.ReportRepository
	metrics      repository.MetricsRepository
	audit        repository.AuditRepository
	expenses     *service.ExpenseService
	suppliers    *service.SupplierService
	purchases    *service.PurchaseService
	factelect    *factelect.FacElectService
	fiscalRepo   factelect.FiscalConfigRepository
	assistant    *service.AssistantService
	tables       *service.DiningTableService
}

// New constructs a Handler wiring up all repositories and services.
func New(db *database.DB) *Handler {
	catRepo := repository.NewCategoryRepository(db.DB)
	prodRepo := repository.NewProductRepository(db.DB)
	saleRepo := repository.NewSaleRepository(db.DB)
	regRepo := repository.NewRegisterRepository(db.DB)
	custRepo := repository.NewCustomerRepository(db.DB)
	userRepo := repository.NewUserRepository(db.DB)
	setupRepo := repository.NewSetupRepository(db.DB)
	stockRepo := repository.NewStockEntryRepository(db.DB)
	reportRepo := repository.NewReportRepository(db.DB)
	metricsRepo := repository.NewMetricsRepository(db.DB)
	auditRepo := repository.NewAuditRepository(db.DB)
	expenseRepo := repository.NewExpenseRepository(db.DB)
	eledocRepo := repository.NewElectronicDocRepository(db.DB)
	fiscalRepo := factelect.NewFiscalConfigRepository(db.DB)
	supplierRepo := repository.NewSupplierRepository(db.DB)
	purchaseRepo := repository.NewPurchaseRepository(db.DB)
	assistantQueryRepo := repository.NewAssistantQueryRepository(db.DB)
	tableRepo := repository.NewDiningTableRepository(db.DB)

	salesSvc := service.NewSaleService(saleRepo, prodRepo, regRepo)
	salesSvc.WithElectronicDocs(eledocRepo)
	registerSvc := service.NewRegisterService(regRepo, userRepo, setupRepo)

	var factelSvc *factelect.FacElectService
	cfg, _ := fiscalRepo.Get(context.Background())
	if cfg != nil && cfg.Enabled {
		client := factelect.NewClient(cfg.ProviderClientID, cfg.ProviderSecret, cfg.Ambiente)
		adapter := factelect.NewPosSaleAdapter(saleRepo, prodRepo, setupRepo, fiscalRepo)
		factelSvc = factelect.NewFacElectService(adapter, client, eledocRepo, fiscalRepo)
		factelSvc.StartPoller(context.Background())
	}

	return &Handler{
		db:           db,
		categories:   service.NewCategoryService(catRepo),
		products:     service.NewProductService(prodRepo),
		sales:        salesSvc,
		registers:    registerSvc,
		customers:    service.NewCustomerService(custRepo),
		users:        service.NewUserService(userRepo),
		setup:        setupRepo,
		stockEntries: stockRepo,
		reports:      reportRepo,
		metrics:      metricsRepo,
		audit:        auditRepo,
		expenses:     service.NewExpenseService(expenseRepo, registerSvc),
		suppliers:    service.NewSupplierService(supplierRepo),
		purchases:    service.NewPurchaseService(purchaseRepo, supplierRepo),
		factelect:    factelSvc,
		fiscalRepo:   fiscalRepo,
		assistant:    service.NewAssistantService(metricsRepo, reportRepo, expenseRepo, assistantQueryRepo),
		tables:       service.NewDiningTableService(tableRepo),
	}
}

// Routes registers all application routes and returns the root handler.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(appmw.CORS)

	// Public
	r.Post("/auth/login", h.login)
	r.Get("/setup/status", h.getSetupStatus)
	r.Post("/onboarding", h.submitOnboarding)
	r.Get("/public/catalog/{token}", h.getPublicCatalog)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(appmw.Auth)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			respond(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Get("/auth/me", h.me)

		// Setup / Business config
		r.Get("/setup", h.getSetup)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Put("/setup", h.updateSetup)
			r.Post("/settings/email/test", h.testEmailConfig)
		})

		// Categories (all authenticated users can read; admin to write)
		r.Get("/categories", h.listCategories)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/categories", h.createCategory)
			r.Put("/categories/{id}", h.updateCategory)
			r.Delete("/categories/{id}", h.deleteCategory)
		})

		// Products (all authenticated users can read; admin to write)
		r.Get("/products", h.listProducts)
		r.Get("/products/low-stock", h.listLowStockProducts)
		r.Get("/products/barcode/{barcode}", h.getProductByBarcode)
			r.Get("/products/{id}", h.getProduct)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/products", h.createProduct)
			r.Put("/products/{id}", h.updateProduct)
			r.Delete("/products/{id}", h.deleteProduct)
			r.Post("/stock-entries", h.createStockEntry)
		})

		// Sales (all authenticated users)
		r.Post("/sales", h.createSale)
		r.Get("/sales", h.listSales)
		r.Get("/sales/{id}", h.getSale)
		r.Get("/sales/{id}/ticket", h.getTicket)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Delete("/sales/{id}/void", h.voidSale)
			r.Post("/sales/{id}/void-with-credit-note", h.voidSaleWithCreditNote)
			r.Get("/sales/{id}/factura", h.getSaleFactura)
			r.Get("/sales/{id}/factura/pdf", h.getSaleFacturaPDF)
		})

		// Fiscal config (admin only)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Get("/fiscal/config", h.getFiscalConfig)
			r.Put("/fiscal/config", h.updateFiscalConfig)
		})

		// Dashboard (admin only)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Get("/dashboard", h.getDashboard)
			r.Get("/dashboard/today", h.getDashboardToday)
		})

		// Reports (admin only)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Get("/reports/portfolio", h.getPortfolio)
			r.Get("/reports/sales", h.getSalesReport)
			r.Get("/reports/top-products", h.getTopProducts)
			r.Get("/reports/registers", h.getRegistersReport)
		})

		// Expenses (all authenticated users)
		r.Post("/expenses", h.createExpense)
		r.Get("/expenses", h.listExpenses)

		// Suppliers (all authenticated users can read; admin to write)
		r.Get("/suppliers", h.listSuppliers)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/suppliers", h.createSupplier)
			r.Put("/suppliers/{id}", h.updateSupplier)
			r.Delete("/suppliers/{id}", h.deleteSupplier)
		})

		// Dining tables (all authenticated users can read; admin to write)
		r.Get("/tables", h.listTables)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/tables", h.createTable)
			r.Put("/tables/{id}", h.updateTable)
			r.Delete("/tables/{id}", h.deleteTable)
		})

		// Purchases (all authenticated users can read; admin to write)
		r.Get("/purchases", h.listPurchases)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/purchases", h.createPurchase)
		})

		// Local assistant (admin only): exposes sales/profit/portfolio figures
		// via chat, same access level as dashboard/reports (it shows the same data).
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/ai/ask", h.askAssistant)
		})

		// Audit log (admin only)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Get("/audit", h.listAudit)
		})

		// Registers (all authenticated users)
		r.Post("/registers", h.openRegister)
		r.Get("/registers", h.listRegisters)
		r.Get("/registers/open", h.getOpenRegister)
		r.Get("/registers/{id}", h.getRegister)
		r.Post("/registers/{id}/close", h.closeRegister)
		r.Get("/registers/{id}/movements", h.listMovements)
		r.Post("/registers/{id}/movements", h.addMovement)

		// Users
		r.Put("/users/me/password", h.changeMyPassword)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Get("/users", h.listUsers)
			r.Post("/users", h.createUser)
			r.Put("/users/{id}", h.updateUser)
			r.Delete("/users/{id}", h.deactivateUser)
		})

		// Customers (all authenticated users can read; admin to write)
		r.Get("/customers", h.listCustomers)
		r.Get("/customers/{id}", h.getCustomer)
		r.Get("/customers/{id}/detail", h.getCustomerDetail)
		r.Get("/customers/{id}/balance", h.getCustomerBalance)
		r.Get("/customers/{id}/payments", h.listCustomerPayments)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/customers", h.createCustomer)
			r.Put("/customers/{id}", h.updateCustomer)
			r.Delete("/customers/{id}", h.deleteCustomer)
			r.Post("/customers/{id}/payments", h.createCustomerPayment)
		})
	})

	return r
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"error": message})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
