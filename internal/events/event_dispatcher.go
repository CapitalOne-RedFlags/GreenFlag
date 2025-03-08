package events

import "github.com/CapitalOne-RedFlags/GreenFlag/internal/models"

type EventDispatcher interface {
	DispatchFraudAlertEvent(transaction models.Transaction) error
}

type GfEventDispatcher struct{}

func (*GfEventDispatcher) DispatchFraudAlertEvent(transaction models.Transaction) error {
	return nil
}
