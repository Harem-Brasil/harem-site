package domain

// Estados do pedido de item do catálogo (creator_orders).
// Fluxo alinhado a billing/webhooks (HB-EPIC-06): requested → awaiting_payment → paid → fulfilled;
// terminais: fulfilled, canceled, refunded.
const (
	OrderStatusRequested        = "requested"
	OrderStatusAwaitingPayment  = "awaiting_payment"
	OrderStatusPaid             = "paid"
	OrderStatusFulfilled        = "fulfilled"
	OrderStatusCanceled         = "canceled"
	OrderStatusRefunded         = "refunded"
)

// Transições válidas (origem → destinos). Regras aplicadas no domínio antes da persistência.
var catalogOrderTransitions = map[string][]string{
	OrderStatusRequested:       {OrderStatusAwaitingPayment, OrderStatusCanceled},
	OrderStatusAwaitingPayment: {OrderStatusPaid, OrderStatusCanceled},
	OrderStatusPaid:            {OrderStatusFulfilled, OrderStatusRefunded},
	// terminais: fulfilled, canceled, refunded — sem saída
}

// IsTerminalCatalogOrderStatus indica estado final.
func IsTerminalCatalogOrderStatus(status string) bool {
	switch status {
	case OrderStatusFulfilled, OrderStatusCanceled, OrderStatusRefunded:
		return true
	default:
		return false
	}
}

// CanTransitionCatalogOrder valida uma única aresta da máquina de estados.
func CanTransitionCatalogOrder(from, to string) bool {
	if from == to {
		return true
	}
	dests, ok := catalogOrderTransitions[from]
	if !ok {
		return false
	}
	for _, d := range dests {
		if d == to {
			return true
		}
	}
	return false
}
