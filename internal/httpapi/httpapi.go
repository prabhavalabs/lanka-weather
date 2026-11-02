// Package httpapi exposes open-weather's read API.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/prabhavalabs/open-weather/internal/layers"
)

// Envelope is the shape of every JSON response.
type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error is a machine-readable error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewRouter returns the API's HTTP handler.
func NewRouter(logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, Envelope{Data: map[string]string{"status": "ok"}})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Get("/layers", listLayers)
		r.Get("/layers/{id}", getLayer)
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	})
	return r
}

func listLayers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: layers.All()})
}

func getLayer(w http.ResponseWriter, r *http.Request) {
	layer, ok := layers.ByID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "layer_not_found", "unknown layer")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, Envelope{Data: layer})
}

func writeJSON(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Envelope{Error: &Error{Code: code, Message: message}})
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
