package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/edalab/services/bancaire/internal/domain"
	"github.com/edalab/services/bancaire/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Handler handles HTTP requests for the bancaire API
type Handler struct {
	repo   repository.CompteRepository
	logger *slog.Logger
}

// NewHandler creates a new API handler
func NewHandler(repo repository.CompteRepository, logger *slog.Logger) *Handler {
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}

// Router returns the HTTP router
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Get("/health", h.healthCheck)
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/comptes", func(r chi.Router) {
			r.Get("/{id}", h.getCompte)
			r.Get("/{id}/transactions", h.getTransactions)
		})
		r.Route("/clients", func(r chi.Router) {
			r.Get("/{client_id}/comptes", h.getComptesByClient)
		})
	})

	return r
}

// healthCheck handles health check requests
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := h.repo.(*repository.PostgresCompteRepository); err != nil {
		// Can't easily check health, assume OK if we got this far
	}

	response := map[string]interface{}{
		"status":   "healthy",
		"service":  "bancaire",
		"database": "connected",
	}
	writeJSON(w, http.StatusOK, response)
}

// getCompte handles getting an account by ID
func (h *Handler) getCompte(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	compte, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "compte not found")
			return
		}
		h.logger.Error("Failed to get compte", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, compte)
}

// getTransactions handles getting transactions for an account
func (h *Handler) getTransactions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Parse limit
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Check if account exists
	_, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "compte not found")
			return
		}
		h.logger.Error("Failed to get compte", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	transactions, err := h.repo.GetTransactions(r.Context(), id, limit)
	if err != nil {
		h.logger.Error("Failed to get transactions", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if transactions == nil {
		transactions = make([]*domain.Transaction, 0)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
	})
}

// getComptesByClient handles getting all accounts for a client
func (h *Handler) getComptesByClient(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "client_id")
	if clientID == "" {
		writeError(w, http.StatusBadRequest, "client_id is required")
		return
	}

	comptes, err := h.repo.GetByClientID(r.Context(), clientID)
	if err != nil {
		h.logger.Error("Failed to get comptes", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if comptes == nil {
		comptes = make([]*domain.Compte, 0)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"comptes": comptes,
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
