package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-xray-sdk-go/xray"
)

func TransactionProcessingHandler(ctx context.Context, event events.SQSEvent, repository db.TransactionRepository) error {
	// Create a new segment for the entire handler
	ctx, seg := xray.BeginSegment(ctx, "TransactionProcessingHandler")
	defer seg.Close(nil)

	// Add metadata about the SQS event
	if err := seg.AddMetadata("EventRecordsCount", len(event.Records)); err != nil {
		fmt.Printf("Failed to add EventRecordsCount metadata: %v\n", err)
	}

	// Create a subsegment for unmarshaling SQS messages
	var transactions []models.Transaction
	ctx, subSeg := xray.BeginSubsegment(ctx, "UnmarshalSQSMessages")
	for i, record := range event.Records {
		// Add annotation for each record
		if err := xray.AddAnnotation(ctx, "MessageID-"+strconv.Itoa(i), record.MessageId); err != nil {
			fmt.Printf("Failed to add MessageID annotation: %v\n", err)
		}

		result, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			if err := subSeg.AddError(err); err != nil {
				fmt.Printf("Failed to add error to subsegment: %v\n", err)
			}
			subSeg.Close(err)
			return err
		}
		transactions = append(transactions, *result)

		// Add transaction metadata to the subsegment
		if err := subSeg.AddMetadata("Transaction-"+strconv.Itoa(i), map[string]interface{}{
			"TransactionID": result.TransactionID,
			"AccountID":     result.AccountID,
			"Amount":        result.TransactionAmount,
			"Email":         result.Email,
		}); err != nil {
			fmt.Printf("Failed to add Transaction metadata: %v\n", err)
		}
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
	if err := txnSeg.AddMetadata("TransactionIDs", transactionIDs); err != nil {
		fmt.Printf("Failed to add TransactionIDs metadata: %v\n", err)
	}
	if err := txnSeg.AddMetadata("AccountIDs", accountIDs); err != nil {
		fmt.Printf("Failed to add AccountIDs metadata: %v\n", err)
	}
	if err := txnSeg.AddMetadata("Emails", emails); err != nil {
		fmt.Printf("Failed to add Emails metadata: %v\n", err)
	}

	// Pass the X-Ray context to the service layer
	err := services.TransactionService(ctx, transactions, repository)
	if err != nil {
		if err := txnSeg.AddError(err); err != nil {
			fmt.Printf("Failed to add error to transaction segment: %v\n", err)
		}
	}
	txnSeg.Close(err)

	return err
}
