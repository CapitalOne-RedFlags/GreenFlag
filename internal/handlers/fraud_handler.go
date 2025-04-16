package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func (fh *GfFraudHandler) ProcessFraudEvent(ctx context.Context, event events.DynamoDBEvent) error {
	// Create a segment for the fraud handler
	ctx, seg := xray.BeginSegment(ctx, "FraudHandler")
	defer seg.Close(nil)

	// Add metadata about the event
	if err := seg.AddMetadata("EventRecordsCount", len(event.Records)); err != nil {
		fmt.Printf("Failed to add EventRecordsCount metadata: %v\n", err)
	}

	// Create a subsegment for processing DynamoDB records
	ctx, procSeg := xray.BeginSubsegment(ctx, "ProcessDynamoDBRecords")

	var transactions []models.Transaction
	var transactionIDs []string

	for i, record := range event.Records {
		// Add annotation for each record
		if err := xray.AddAnnotation(ctx, "RecordID-"+strconv.Itoa(i), record.EventID); err != nil {
			fmt.Printf("Failed to add RecordID annotation: %v\n", err)
		}

		if record.EventName != "INSERT" {
			continue
		}

		attributeValueMap := make(map[string]types.AttributeValue)
		for attr, dynamoAttributeValue := range record.Change.NewImage {
			attributeValueMap[attr] = convertDynamoDBAttributeValue(dynamoAttributeValue)
		}

		transaction, err := models.UnmarshalDynamoDB(attributeValueMap)
		if err != nil {
			if err := procSeg.AddError(err); err != nil {
				fmt.Printf("Failed to add error to segment: %v\n", err)
			}
			procSeg.Close(err)
			return err
		}

		transactions = append(transactions, *transaction)
		transactionIDs = append(transactionIDs, transaction.TransactionID)
	}

	// Add transaction IDs to metadata
	if err := procSeg.AddMetadata("TransactionIDs", transactionIDs); err != nil {
		fmt.Printf("Failed to add TransactionIDs metadata: %v\n", err)
	}
	procSeg.Close(nil)

	// Create a subsegment for fraud prediction
	_, fraudSeg := xray.BeginSubsegment(ctx, "FraudPrediction")

	// Track emails for potential fraud
	var emailsChecked []string
	for _, txn := range transactions {
		emailsChecked = append(emailsChecked, txn.Email)
	}
	if err := fraudSeg.AddMetadata("EmailsChecked", emailsChecked); err != nil {
		fmt.Printf("Failed to add EmailsChecked metadata: %v\n", err)
	}

	// Call the fraud service
	fraudulentTransactions, err := fh.FraudService.PredictFraud(transactions)

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

		if err := fraudSeg.AddMetadata("FraudDetected", true); err != nil {
			fmt.Printf("Failed to add FraudDetected metadata: %v\n", err)
		}
		if err := fraudSeg.AddMetadata("FraudulentTransactionIDs", fraudIDs); err != nil {
			fmt.Printf("Failed to add FraudulentTransactionIDs metadata: %v\n", err)
		}
		if err := fraudSeg.AddMetadata("FraudulentEmails", fraudEmails); err != nil {
			fmt.Printf("Failed to add FraudulentEmails metadata: %v\n", err)
		}
		if err := fraudSeg.AddMetadata("FraudulentAmounts", fraudAmounts); err != nil {
			fmt.Printf("Failed to add FraudulentAmounts metadata: %v\n", err)
		}
		if err := fraudSeg.AddMetadata("FraudCount", len(fraudulentTransactions)); err != nil {
			fmt.Printf("Failed to add FraudCount metadata: %v\n", err)
		}
	} else {
		if err := fraudSeg.AddMetadata("FraudDetected", false); err != nil {
			fmt.Printf("Failed to add FraudDetected metadata: %v\n", err)
		}
		if err := fraudSeg.AddMetadata("FraudCount", 0); err != nil {
			fmt.Printf("Failed to add FraudCount metadata: %v\n", err)
		}
	}

	if err != nil {
		if err := fraudSeg.AddError(err); err != nil {
			fmt.Printf("Failed to add error to fraud segment: %v\n", err)
		}
	}

	fraudSeg.Close(err)
	return err
}

// Converts AWS Lambda event DynamoDBAttributeValue to AWS SDK v2 AttributeValue
func convertDynamoDBAttributeValue(attr events.DynamoDBAttributeValue) types.AttributeValue {
	switch attr.DataType() {
	case events.DataTypeString:
		return &types.AttributeValueMemberS{Value: attr.String()}
	case events.DataTypeNumber:
		return &types.AttributeValueMemberN{Value: attr.Number()}
	case events.DataTypeBoolean:
		return &types.AttributeValueMemberBOOL{Value: attr.Boolean()}
	case events.DataTypeBinary:
		return &types.AttributeValueMemberB{Value: attr.Binary()}
	case events.DataTypeNull:
		return &types.AttributeValueMemberNULL{Value: attr.IsNull()}
	case events.DataTypeStringSet:
		return &types.AttributeValueMemberSS{Value: attr.StringSet()}
	case events.DataTypeNumberSet:
		return &types.AttributeValueMemberNS{Value: attr.NumberSet()}
	case events.DataTypeBinarySet:
		return &types.AttributeValueMemberBS{Value: attr.BinarySet()}
	case events.DataTypeMap:
		mapped := make(map[string]types.AttributeValue)
		for k, v := range attr.Map() {
			mapped[k] = convertDynamoDBAttributeValue(v)
		}
		return &types.AttributeValueMemberM{Value: mapped}
	case events.DataTypeList:
		var list []types.AttributeValue
		for _, v := range attr.List() {
			list = append(list, convertDynamoDBAttributeValue(v))
		}
		return &types.AttributeValueMemberL{Value: list}
	default:
		fmt.Printf("Unsupported attribute type: %v\n", attr.DataType())
		return nil
	}
}
