package handlers

import (
	"context"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

func TransactionProcessingHandler(ctx context.Context, event events.SQSEvent, repository db.TransactionRepository) error {

	var transactions []models.Transaction
	for _, record := range event.Records {
		result, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			return err
		}
		transactions = append(transactions, *result)
	}
	return services.TransactionService(ctx, transactions, repository)
}
