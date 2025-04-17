package handlers

import (
	"context"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

type FraudRetryHandler struct {
	FraudService services.FraudService
}

func NewFraudRetryHandler(fraudService services.FraudService) *FraudRetryHandler {
	return &FraudRetryHandler{
		FraudService: fraudService,
	}
}

func (frh *FraudRetryHandler) ProcessDLQFraudEvent(ctx context.Context, event events.SQSEvent) (*models.BatchResult, error) {
	var errorResults []error
	var failedRIDs []string
	var transactions []models.Transaction
	messageIdsByTransactionId := make(map[string]string)

	for _, record := range event.Records {
		txn, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			errorResults = append(errorResults, err)
			failedRIDs = append(failedRIDs, record.MessageId)
		}

		transactions = append(transactions, *txn)
		messageIdsByTransactionId[txn.TransactionID] = record.MessageId
	}

	_, failedTransactions, err := frh.FraudService.PredictFraud(transactions)
	if err != nil {
		errorResults = append(errorResults, err)
	}

	batchResultInput := &middleware.GetBatchResultInput{
		FailedTransactions:  failedTransactions,
		RIDsByTransactionId: messageIdsByTransactionId,
		FailedRIDs:          failedRIDs,
		Errors:              errorResults,
	}

	return middleware.GetBatchResult(batchResultInput)
}
