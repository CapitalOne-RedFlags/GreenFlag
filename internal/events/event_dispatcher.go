package events

import (
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type EventDispatcher interface {
	DispatchFraudAlertEvent(transaction models.Transaction) error
	DispatchFraudUpdateEvent(number string, body string) error
}

type GfEventDispatcher struct {
	SNSMessenger messaging.SNSMessenger
}

func NewGfEventDispatcher(snsMessenger messaging.SNSMessenger) *GfEventDispatcher {
	return &GfEventDispatcher{
		SNSMessenger: snsMessenger,
	}
}

// To be refactored later
func (dispatcher *GfEventDispatcher) DispatchFraudAlertEvent(transaction models.Transaction) error {

	// _, err := dispatcher.SNSMessenger.SendEmailAlert(transaction)
	err := dispatcher.SNSMessenger.SendTextAlert(transaction)
	if err != nil {
		return fmt.Errorf("error sending text message for transaction: %s", err)
	}
	fmt.Printf("Fraud detected, successfully sent text to %s\n", transaction.PhoneNumber)

	return nil
}

func (dispatcher *GfEventDispatcher) DispatchFraudUpdateEvent(number string, body string) error {
	err := dispatcher.SNSMessenger.SendTextUpdate(number, body)
	if err != nil {
		return fmt.Errorf("error sending text message for transaction: %s", err)
	}
	fmt.Printf("Fraud event updated: successfully sent replied to %s\n", number)
	return nil
}
