package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/harem-brasil/backend/internal/domain"
)

var validWebhookProviders = map[string]bool{
	"stripe":      true,
	"pagseguro":   true,
	"mercadopago": true,
}

func (s *Services) validateWebhookSignature(provider string, body []byte, sigHeader, stripeSig string) bool {
	sig := sigHeader
	if sig == "" {
		sig = stripeSig
	}

	var secretKey string
	switch provider {
	case "stripe":
		secretKey = s.StripeWebhookSecret
	case "pagseguro":
		secretKey = s.PagSeguroWebhookSecret
	case "mercadopago":
		secretKey = s.MercadoPagoWebhookSecret
	default:
		return false
	}

	// Sem segredo: só development/test aceitam (ex.: testes BDD sem header de assinatura).
	if secretKey == "" {
		return s.AppEnv == "development" || s.AppEnv == "test"
	}

	if sig == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

func (s *Services) WebhookStripe(ctx context.Context, body []byte, sigHeader, stripeSig string) error {
	if !s.validateWebhookSignature("stripe", body, sigHeader, stripeSig) {
		return domain.Err(401, "Invalid webhook signature")
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		return domain.Err(400, "Invalid JSON")
	}

	return nil
}

func (s *Services) WebhookPagSeguro(ctx context.Context, body []byte, sigHeader string) error {
	if !s.validateWebhookSignature("pagseguro", body, sigHeader, "") {
		return domain.Err(401, "Invalid webhook signature")
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		return domain.Err(400, "Invalid JSON")
	}

	return nil
}

func (s *Services) WebhookMercadoPago(ctx context.Context, body []byte, sigHeader string) error {
	if !s.validateWebhookSignature("mercadopago", body, sigHeader, "") {
		return domain.Err(401, "Invalid webhook signature")
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		return domain.Err(400, "Invalid JSON")
	}

	return nil
}

func (s *Services) WebhookGeneric(ctx context.Context, provider string, body []byte, sigHeader string) (map[string]string, error) {
	if !validWebhookProviders[provider] {
		return nil, domain.Err(404, "Unknown webhook provider")
	}

	if !s.validateWebhookSignature(provider, body, sigHeader, "") {
		return nil, domain.Err(401, "Invalid webhook signature")
	}

	var event map[string]any
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, domain.Err(400, "Invalid JSON")
	}

	eventID, ok1 := event["id"].(string)
	eventType, ok2 := event["type"].(string)
	if !ok1 || !ok2 {
		return nil, domain.Err(400, "Missing required fields: id, type")
	}

	s.Logger.Info("webhook received",
		"provider", provider,
		"event_id", eventID,
		"event_type", eventType,
	)

	return map[string]string{
		"status":   "received",
		"provider": provider,
		"event_id": eventID,
	}, nil
}
