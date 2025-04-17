package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/observability"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type TransactionProcessingHandler struct {
	service services.TransactionService
}

func NewTransactionProcessingHandler(service services.TransactionService) *TransactionProcessingHandler {
	return &TransactionProcessingHandler{
		service: service,
	}
}

func (tph *TransactionProcessingHandler) TransactionProcessingHandler(ctx context.Context, event events.SQSEvent) (*models.BatchResult, error) {
	var errorResults []error
	var failedRIDs []string

	// Create a new segment for the entire handler
	ctx, seg := xray.BeginSegment(ctx, "TransactionProcessingHandler")
	defer seg.Close(nil)

	// Add metadata about the SQS event
	observability.SafeAddMetadata(seg, observability.KeyEventRecordsCount, len(event.Records))

	var transactions []models.Transaction
	messageIdsByTransactionId := make(map[string]string)

	// Create a subsegment for unmarshaling SQS messages
	ctx, subSeg := xray.BeginSubsegment(ctx, "UnmarshalSQSMessages")

	for i, record := range event.Records {
		// Add annotation for each record
		observability.SafeAddAnnotation(ctx, "MessageID-"+strconv.Itoa(i), record.MessageId)

		transaction, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			fmt.Printf("error Unmarshalling: %s", err)
			errorResults = append(errorResults, err)
			failedRIDs = append(failedRIDs, record.MessageId)

			observability.SafeAddError(subSeg, err)
			subSeg.Close(err)

			continue
		}

		transactions = append(transactions, *transaction)
		messageIdsByTransactionId[transaction.TransactionID] = record.MessageId

		// Add transaction metadata to the subsegment
		observability.SafeAddMetadata(subSeg, observability.KeyTransaction+strconv.Itoa(i), map[string]interface{}{
			"TransactionID": transaction.TransactionID,
			"AccountID":     transaction.AccountID,
			"Amount":        transaction.TransactionAmount,
			"Email":         transaction.Email,
		})
	}
	subSeg.Close(nil)

	// Create a subsegment for transaction processing
	ctx, txnSeg := xray.BeginSubsegment(ctx, "TransactionService")

	// Add transaction IDs to metadata for easier searching
	var transactionIDs []string
	var accountIDs []string
	var emails []string
	for _, txn := range transactions {
		transactionIDs = append(transactionIDs, txn.TransactionID)
		accountIDs = append(accountIDs, txn.AccountID)
		emails = append(emails, txn.Email)
	}
	observability.SafeAddMetadata(txnSeg, observability.KeyTransactionIDs, transactionIDs)
	observability.SafeAddMetadata(txnSeg, observability.KeyAccountIDs, accountIDs)
	observability.SafeAddMetadata(txnSeg, observability.KeyEmails, emails)

	// Pass the X-Ray context to the service layer
	failedTransactions, err := tph.service.TransactionService(ctx, transactions)
	if err != nil {
		errorResults = append(errorResults, err)
		observability.SafeAddError(txnSeg, err)
	}

	txnSeg.Close(err)

	batchResultInput := &middleware.GetBatchResultInput{
		FailedTransactions:  failedTransactions,
		RIDsByTransactionId: messageIdsByTransactionId,
		FailedRIDs:          failedRIDs,
		Errors:              errorResults,
	}

	return middleware.GetBatchResult(batchResultInput)
}
