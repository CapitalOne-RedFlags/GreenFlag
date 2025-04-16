package handlers

import (
	"context"

	gfEvents "github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

type TransactionProcessingHandler struct {
	service services.TransactionService
}

func NewTransactionProcessingHandler(service *services.TransactionService) *TransactionProcessingHandler {
	return &TransactionProcessingHandler{
		service: *service,
	}
}

func (tph *TransactionProcessingHandler) TransactionProcessingHandler(ctx context.Context, event events.SQSEvent) (*gfEvents.BatchResult, error) {
	var errorResults []error
	var failedRIDs []string
	var transactions []models.Transaction
	messageIdsByTransactionId := make(map[string]string)

	for _, record := range event.Records {
		transaction, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			errorResults = append(errorResults, err)
			failedRIDs = append(failedRIDs, record.MessageId)
		}

		transactions = append(transactions, *transaction)
		messageIdsByTransactionId[transaction.TransactionID] = record.MessageId
	}

	failedTransactions, err := tph.service.TransactionService(ctx, transactions)
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
