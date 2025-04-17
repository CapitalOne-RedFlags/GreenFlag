package handlers

import (
	"context"
	"strconv"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/observability"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-xray-sdk-go/xray"
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

func (fh *GfFraudHandler) ProcessFraudEvent(ctx context.Context, event events.DynamoDBEvent) (*models.BatchResult, error) {
	var errorResults []error
	var failedRIDs []string

	// Create a segment for the fraud handler
	ctx, seg := xray.BeginSegment(ctx, "FraudHandler")
	defer seg.Close(nil)

	// Add metadata about the event
	observability.SafeAddMetadata(seg, observability.KeyEventRecordsCount, len(event.Records))

	// Create a subsegment for processing DynamoDB records
	ctx, procSeg := xray.BeginSubsegment(ctx, "ProcessDynamoDBRecords")

	var transactions []models.Transaction
	ridsByTransactionId := make(map[string]string)
	var transactionIDs []string

	for i, record := range event.Records {
		// Add annotation for each record
		observability.SafeAddAnnotation(ctx, "RecordID-"+strconv.Itoa(i), record.EventID)

		if record.EventName != "INSERT" {
			continue
		}

		transaction, err := models.UnmarshalStreamImage(record.Change.NewImage)
		if err != nil {
			errorResults = append(errorResults, err)
			failedRIDs = append(failedRIDs, record.Change.SequenceNumber)

			observability.SafeAddError(procSeg, err)
			procSeg.Close(err)

			continue
		}

		transactions = append(transactions, *transaction)
		transactionIDs = append(transactionIDs, transaction.TransactionID)
		ridsByTransactionId[transaction.TransactionID] = record.Change.SequenceNumber
	}

	// Add transaction IDs to metadata
	observability.SafeAddMetadata(procSeg, observability.KeyTransactionIDs, transactionIDs)
	procSeg.Close(nil)

	// Create a subsegment for fraud prediction
	_, fraudSeg := xray.BeginSubsegment(ctx, "FraudPrediction")

	// Track emails for potential fraud
	var emailsChecked []string
	for _, txn := range transactions {
		emailsChecked = append(emailsChecked, txn.Email)
	}
	observability.SafeAddMetadata(fraudSeg, observability.KeyEmailsChecked, emailsChecked)

	// Call the fraud service
	fraudulentTransactions, failedTransactions, err := fh.FraudService.PredictFraud(ctx, transactions)

	// Add metadata about fraudulent transactions
	if len(fraudulentTransactions) > 0 {
		var fraudIDs []string
		var fraudEmails []string
		var fraudAmounts []float64

		for _, txn := range fraudulentTransactions {
			fraudIDs = append(fraudIDs, txn.TransactionID)
			fraudEmails = append(fraudEmails, txn.Email)
			fraudAmounts = append(fraudAmounts, txn.TransactionAmount)
		}

		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudDetected, true)
		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudulentTransactionIDs, fraudIDs)
		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudulentEmails, fraudEmails)
		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudulentAmounts, fraudAmounts)
		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudCount, len(fraudulentTransactions))
	} else {
		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudDetected, false)
		observability.SafeAddMetadata(fraudSeg, observability.KeyFraudCount, 0)
	}

	if err != nil {
		errorResults = append(errorResults, err)
		observability.SafeAddError(fraudSeg, err)
	}

	fraudSeg.Close(err)

	batchResultInput := &middleware.GetBatchResultInput{
		FailedTransactions:  failedTransactions,
		RIDsByTransactionId: ridsByTransactionId,
		FailedRIDs:          failedRIDs,
		Errors:              errorResults,
	}

	return middleware.GetBatchResult(batchResultInput)
}
