package handlers

import (
	"context"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	var transactions []models.Transaction
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		attributeValueMap := make(map[string]types.AttributeValue)
		for attr, dynamoAttributeValue := range record.Change.NewImage {
			attributeValueMap[attr] = convertDynamoDBAttributeValue(dynamoAttributeValue)
		}

		transaction, err := models.UnmarshalDynamoDB(attributeValueMap)
		if err != nil {
			return err
		}

		transactions = append(transactions, *transaction)
	}

	return fh.FraudService.PredictFraud(transactions)
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
