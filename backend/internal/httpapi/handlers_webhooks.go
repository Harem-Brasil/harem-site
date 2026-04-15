package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type WebhookEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Provider  string         `json:"provider"`
	Data      map[string]any `json:"data"`
	CreatedAt string         `json:"created_at"`
}

func (s *Server) handleWebhookStripe(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// TODO: Validate Stripe signature with HMAC
	// TODO: Check idempotency by event ID
	// TODO: Queue event for async processing

	// Return 200 quickly as per spec
	respondJSON(w, map[string]string{"status": "received"})
}

func (s *Server) handleWebhookPagSeguro(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// TODO: Validate PagSeguro signature with HMAC
	// TODO: Check idempotency by event ID
	// TODO: Queue event for async processing

	respondJSON(w, map[string]string{"status": "received"})
}

func (s *Server) handleWebhookMercadoPago(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// TODO: Validate MercadoPago signature with HMAC
	// TODO: Check idempotency by event ID
	// TODO: Queue event for async processing

	respondJSON(w, map[string]string{"status": "received"})
}

func (s *Server) handleWebhookGeneric(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	validProviders := map[string]bool{
		"stripe":      true,
		"pagseguro":   true,
		"mercadopago": true,
	}

	if !validProviders[provider] {
		respondError(w, http.StatusNotFound, "Unknown webhook provider")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Log without full payload for security (only event ID and type)
	eventID, _ := event["id"].(string)
	eventType, _ := event["type"].(string)

	s.config.Logger.Info("webhook received",
		"provider", provider,
		"event_id", eventID,
		"event_type", eventType,
	)

	// Return 200 quickly - processing is async
	respondJSON(w, map[string]string{
		"status":   "received",
		"provider": provider,
		"event_id": eventID,
	})
}
