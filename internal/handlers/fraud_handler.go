package handlers

import (
	"context"

	GFEvents "github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
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

func (fh *GfFraudHandler) ProcessFraudEvent(ctx context.Context, event events.DynamoDBEvent) (*GFEvents.BatchResult, error) {
	var errorResults []error
	var failedRIDs []string
	var transactions []models.Transaction
	ridsByTransactionId := make(map[string]string)

	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		transaction, err := models.UnmarshalStreamImage(record.Change.NewImage)
		if err != nil {
			errorResults = append(errorResults, err)
			failedRIDs = append(failedRIDs, record.Change.SequenceNumber)
		}

		transactions = append(transactions, *transaction)
		ridsByTransactionId[transaction.TransactionID] = record.Change.SequenceNumber
	}

	failedTransactions, err := fh.FraudService.PredictFraud(transactions)
	if err != nil {
		errorResults = append(errorResults, err)
	}

	batchResultInput := &middleware.GetBatchResultInput{
		FailedTransactions:  failedTransactions,
		RIDsByTransactionId: ridsByTransactionId,
		FailedRIDs:          failedRIDs,
		Errors:              errorResults,
	}

	return middleware.GetBatchResult(batchResultInput)
}
