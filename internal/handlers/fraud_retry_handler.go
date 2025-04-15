package handlers

import (
	"context"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

type FraudRetryHandler struct {
	service services.FraudService
}

func NewFraudRetryHandler(fraudService services.FraudService) *FraudRetryHandler {
	return &FraudRetryHandler{
		service: fraudService,
	}
}

func (frh *FraudRetryHandler) ProcessDLQFraudEvent(ctx context.Context, event events.SQSEvent) error {
	var transactions []models.Transaction
	for _, record := range event.Records {
		txn, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			return err
		}

		transactions = append(transactions, *txn)
	}

	return frh.service.PredictFraud(transactions)
}
