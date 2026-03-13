package handler

import (
	"encoding/json"
	"net/http"

	"github.com/andreyo/pos/internal/database"
	appmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Handler holds all HTTP handler methods and their dependencies.
type Handler struct {
	db         *database.DB
	categories *service.CategoryService
	products   *service.ProductService
	setup      *repository.SetupRepository
}

// New constructs a Handler wiring up all repositories and services.
func New(db *database.DB) *Handler {
	catRepo := repository.NewCategoryRepository(db.DB)
	prodRepo := repository.NewProductRepository(db.DB)
	setupRepo := repository.NewSetupRepository(db.DB)
	return &Handler{
		db:         db,
		categories: service.NewCategoryService(catRepo),
		products:   service.NewProductService(prodRepo),
		setup:      setupRepo,
	}
}

// Routes registers all application routes and returns the root handler.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(appmw.CORS)

	// Public
	r.Post("/auth/login", h.login)

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
		r.Get("/products/{id}", h.getProduct)
		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAdmin)
			r.Post("/products", h.createProduct)
			r.Put("/products/{id}", h.updateProduct)
			r.Delete("/products/{id}", h.deleteProduct)
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
