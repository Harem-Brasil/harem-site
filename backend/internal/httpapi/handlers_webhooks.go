package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

var validWebhookProviders = map[string]bool{
	"stripe":      true,
	"pagseguro":   true,
	"mercadopago": true,
}

func validateWebhookSignature(provider string, body []byte, r *http.Request) bool {
	sigHeader := r.Header.Get("X-Signature")
	if sigHeader == "" {
		sigHeader = r.Header.Get("Stripe-Signature")
	}
	if sigHeader == "" {
		return false
	}

	var secretKey string
	switch provider {
	case "stripe":
		secretKey = os.Getenv("STRIPE_WEBHOOK_SECRET")
	case "pagseguro":
		secretKey = os.Getenv("PAGSEGURO_WEBHOOK_SECRET")
	case "mercadopago":
		secretKey = os.Getenv("MERCADOPAGO_WEBHOOK_SECRET")
	default:
		return false
	}

	if secretKey == "" {
		if os.Getenv("ENV") == "development" || os.Getenv("ENV") == "test" {
			return true
		}
		return false
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sigHeader), []byte(expectedSig))
}

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

	if !validateWebhookSignature("stripe", body, r) {
		respondError(w, http.StatusUnauthorized, "Invalid webhook signature")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// TODO: Check idempotency by event ID
	// TODO: Queue event for async processing

	respondJSON(w, map[string]string{"status": "received"})
}

func (s *Server) handleWebhookPagSeguro(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	if !validateWebhookSignature("pagseguro", body, r) {
		respondError(w, http.StatusUnauthorized, "Invalid webhook signature")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

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

	if !validateWebhookSignature("mercadopago", body, r) {
		respondError(w, http.StatusUnauthorized, "Invalid webhook signature")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// TODO: Check idempotency by event ID
	// TODO: Queue event for async processing

	respondJSON(w, map[string]string{"status": "received"})
}

func (s *Server) handleWebhookGeneric(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	if !validWebhookProviders[provider] {
		respondError(w, http.StatusNotFound, "Unknown webhook provider")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	if !validateWebhookSignature(provider, body, r) {
		respondError(w, http.StatusUnauthorized, "Invalid webhook signature")
		return
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	eventID, ok1 := event["id"].(string)
	eventType, ok2 := event["type"].(string)
	if !ok1 || !ok2 {
		respondError(w, http.StatusBadRequest, "Missing required fields: id, type")
		return
	}

	s.config.Logger.Info("webhook received",
		"provider", provider,
		"event_id", eventID,
		"event_type", eventType,
	)

	respondJSON(w, map[string]string{
		"status":   "received",
		"provider": provider,
		"event_id": eventID,
	})
}
