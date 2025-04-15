package handlers

import (
	"context"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
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

func (tph *TransactionProcessingHandler) TransactionProcessingHandler(ctx context.Context, event events.SQSEvent, repository db.TransactionRepository) (*gfEvents.BatchResult, error) {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(event.Records))
	batchResults := make(chan gfEvents.BatchItemFailure, len(event.Records))

	for _, record := range event.Records {
		transaction, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			errorResults <- err
			batchResults <- gfEvents.BatchItemFailure{
				ItemIdentifier: record.MessageId,
			}
			continue
		}

		if err := tph.service.SaveTransaction(ctx, *transaction, &wg); err != nil {
			errorResults <- err
			batchResults <- gfEvents.BatchItemFailure{
				ItemIdentifier: record.MessageId,
			}
		}
	}

	close(errorResults)
	close(batchResults)

	return middleware.MergeBatchItemFailures(batchResults), middleware.MergeErrors(errorResults)
}
