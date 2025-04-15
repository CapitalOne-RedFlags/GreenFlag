package handlers

import (
	"context"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

type FraudHandler interface {
	ProcessFraudEvent(ctx context.Context, event events.DynamoDBEvent) error
}

type GfFraudHandler struct {
	FraudService services.FraudService
}

func NewFraudHandler(fraudService services.FraudService) *GfFraudHandler {
	return &GfFraudHandler{
		FraudService: fraudService,
	}
}

func (fh *GfFraudHandler) ProcessFraudEvent(ctx context.Context, event events.DynamoDBEvent) error {
	var transactions []models.Transaction
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		transaction, err := models.UnmarshalStreamImage(record.Change.NewImage)
		if err != nil {
			return err
		}

		transactions = append(transactions, *transaction)
	}

	return fh.FraudService.PredictFraud(transactions)
}
