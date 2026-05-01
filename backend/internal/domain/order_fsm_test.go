package domain

import "testing"

func TestCanTransitionCatalogOrder_table(t *testing.T) {
	tests := []struct {
		from, to string
		want     bool
	}{
		// requested
		{OrderStatusRequested, OrderStatusRequested, true},
		{OrderStatusRequested, OrderStatusAwaitingPayment, true},
		{OrderStatusRequested, OrderStatusCanceled, true},
		{OrderStatusRequested, OrderStatusPaid, false},
		{OrderStatusRequested, OrderStatusFulfilled, false},
		{OrderStatusRequested, OrderStatusRefunded, false},

		// awaiting_payment
		{OrderStatusAwaitingPayment, OrderStatusAwaitingPayment, true},
		{OrderStatusAwaitingPayment, OrderStatusPaid, true},
		{OrderStatusAwaitingPayment, OrderStatusCanceled, true},
		{OrderStatusAwaitingPayment, OrderStatusFulfilled, false},
		{OrderStatusAwaitingPayment, OrderStatusRefunded, false},

		// paid
		{OrderStatusPaid, OrderStatusPaid, true},
		{OrderStatusPaid, OrderStatusFulfilled, true},
		{OrderStatusPaid, OrderStatusRefunded, true},
		{OrderStatusPaid, OrderStatusCanceled, false},
		{OrderStatusPaid, OrderStatusAwaitingPayment, false},

		// terminais — apenas noop
		{OrderStatusFulfilled, OrderStatusFulfilled, true},
		{OrderStatusFulfilled, OrderStatusPaid, false},
		{OrderStatusFulfilled, OrderStatusRefunded, false},
		{OrderStatusCanceled, OrderStatusCanceled, true},
		{OrderStatusCanceled, OrderStatusPaid, false},
		{OrderStatusRefunded, OrderStatusRefunded, true},
		{OrderStatusRefunded, OrderStatusPaid, false},

		// desconhecido
		{"unknown", OrderStatusPaid, false},
	}
	for _, tt := range tests {
		got := CanTransitionCatalogOrder(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("CanTransitionCatalogOrder(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestIsTerminalCatalogOrderStatus(t *testing.T) {
	for _, s := range []string{OrderStatusFulfilled, OrderStatusCanceled, OrderStatusRefunded} {
		if !IsTerminalCatalogOrderStatus(s) {
			t.Errorf("expected terminal: %s", s)
		}
	}
	for _, s := range []string{OrderStatusRequested, OrderStatusAwaitingPayment, OrderStatusPaid} {
		if IsTerminalCatalogOrderStatus(s) {
			t.Errorf("expected non-terminal: %s", s)
		}
	}
}
