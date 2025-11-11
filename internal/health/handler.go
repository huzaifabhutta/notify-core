package health

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

// Handler handles HTTP health check requests
type Handler struct {
	checker *Checker
	logger  zerolog.Logger
}

// NewHandler creates a new health check HTTP handler
func NewHandler(checker *Checker, logger zerolog.Logger) *Handler {
	return &Handler{
		checker: checker,
		logger:  logger.With().Str("handler", "health").Logger(),
	}
}

// ServeHTTP handles health check HTTP requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Perform health check
	report := h.checker.Check(r.Context())

	// Determine HTTP status code based on health status
	statusCode := http.StatusOK
	switch report.Status {
	case StatusHealthy:
		statusCode = http.StatusOK
	case StatusDegraded:
		statusCode = http.StatusOK // Still return 200 for degraded
	case StatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case StatusUnknown:
		statusCode = http.StatusServiceUnavailable
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(statusCode)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(report); err != nil {
		h.logger.Error().
			Err(err).
			Msg("Failed to encode health check response")
	}

	h.logger.Debug().
		Str("status", string(report.Status)).
		Int("http_status", statusCode).
		Str("remote_addr", r.RemoteAddr).
		Msg("Health check request completed")
}
