package handlers

import (
	"context"
	"strconv"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/observability"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-xray-sdk-go/xray"
)

func TransactionProcessingHandler(ctx context.Context, event events.SQSEvent, repository db.TransactionRepository) error {
	// Create a new segment for the entire handler
	ctx, seg := xray.BeginSegment(ctx, "TransactionProcessingHandler")
	defer seg.Close(nil)

	// Add metadata about the SQS event
	observability.SafeAddMetadata(seg, observability.KeyEventRecordsCount, len(event.Records))

	// Create a subsegment for unmarshaling SQS messages
	var transactions []models.Transaction
	ctx, subSeg := xray.BeginSubsegment(ctx, "UnmarshalSQSMessages")
	for i, record := range event.Records {
		// Add annotation for each record
		observability.SafeAddAnnotation(ctx, "MessageID-"+strconv.Itoa(i), record.MessageId)

		result, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			observability.SafeAddError(subSeg, err)
			subSeg.Close(err)
			return err
		}
		transactions = append(transactions, *result)

		// Add transaction metadata to the subsegment
		observability.SafeAddMetadata(subSeg, observability.KeyTransaction+strconv.Itoa(i), map[string]interface{}{
			"TransactionID": result.TransactionID,
			"AccountID":     result.AccountID,
			"Amount":        result.TransactionAmount,
			"Email":         result.Email,
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
	err := services.TransactionService(ctx, transactions, repository)
	if err != nil {
		observability.SafeAddError(txnSeg, err)
	}
	txnSeg.Close(err)

	return err
}
