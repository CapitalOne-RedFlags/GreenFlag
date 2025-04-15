package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator"
)

// Transaction represents a record in DynamoDB.
type Transaction struct {
	TransactionID           string  `json:"transactionId" dynamodbav:"TransactionID" validate:"required"`
	AccountID               string  `json:"accountId" dynamodbav:"AccountID" validate:"required"`
	TransactionAmount       float64 `json:"amount" dynamodbav:"TransactionAmount" validate:"gte=0"`
	TransactionDate         string  `json:"transactionDate" dynamodbav:"TransactionDate"`
	TransactionType         string  `json:"transactionType" dynamodbav:"TransactionType"`
	Location                string  `json:"location" dynamodbav:"Location"`
	DeviceID                string  `json:"deviceId" dynamodbav:"DeviceID"`
	IPAddress               string  `json:"ipAddress" dynamodbav:"IPAddress"`
	MerchantID              string  `json:"merchantId" dynamodbav:"MerchantID"`
	Channel                 string  `json:"channel" dynamodbav:"Channel"`
	CustomerAge             int     `json:"customerAge" dynamodbav:"CustomerAge" validate:"gte=18"`
	CustomerOccupation      string  `json:"customerOccupation" dynamodbav:"CustomerOccupation"`
	TransactionDuration     int     `json:"transactionDuration" dynamodbav:"TransactionDuration"`
	LoginAttempts           int     `json:"loginAttempts" dynamodbav:"LoginAttempts"`
	AccountBalance          float64 `json:"accountBalance" dynamodbav:"AccountBalance"`
	PreviousTransactionDate string  `json:"previousTransactionDate" dynamodbav:"PreviousTransactionDate"`
	PhoneNumber             string  `json:"phoneNumber" dynamodbav:"PhoneNumber" validate:"required,e164"`
	Email                   string  `json:"email" dynamodbav:"Email" validate:"required,email"`
	TransactionStatus       string  `json:"transactionStatus" dynamodbav:"TransactionStatus"`
}

// MarshalDynamoDB marshals a Transaction into a DynamoDB attribute map.
func (t *Transaction) MarshalDynamoDB() (map[string]types.AttributeValue, error) {
	return attributevalue.MarshalMap(t)
}

// UnmarshalDynamoDB unmarshals a DynamoDB attribute map into a Transaction.
func UnmarshalDynamoDB(av map[string]types.AttributeValue) (*Transaction, error) {
	var trans Transaction
	if err := attributevalue.UnmarshalMap(av, &trans); err != nil {
		return nil, err
	}
	return &trans, nil
}

func UnmarshalStreamImage(streamImage map[string]events.DynamoDBAttributeValue) (*Transaction, error) {
	attributeValueMap := make(map[string]types.AttributeValue)
	for attr, dynamoAttributeValue := range streamImage {
		attributeValueMap[attr] = convertDynamoDBAttributeValue(dynamoAttributeValue)
	}

	transaction, err := UnmarshalDynamoDB(attributeValueMap)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func UnmarshalSQS(trasaction string) (*Transaction, error) {
	var result Transaction
	err := json.Unmarshal([]byte(trasaction), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil

}

// ValidateTransaction validates an incoming transaction.
func (t *Transaction) ValidateTransaction() error {
	validate := validator.New()
	return validate.Struct(t)
}

// TransactionUpdatePayload builds a DynamoDB update map by
// 1) Skipping the primary keys (TransactionID, AccountID).
// 2) Including only allowed fields.
// 3) Excluding empty/zero fields.
//
// This prevents overwriting non-updated fields with zero values.
// TransactionUpdatePayload returns only the plain typed values that changed.
// We do NOT store types.AttributeValue here, let the UpdateItem code handle marshaling.

func (t *Transaction) TransactionUpdatePayload() (map[string]interface{}, error) {
	updateMap := make(map[string]interface{})

	// Convert struct -> attributevalue map, but then convert each field back to a Go type
	// or skip it if empty, skip if key is disallowed, etc.
	avMap, err := attributevalue.MarshalMap(t)
	if err != nil {
		return nil, err
	}

	for field, av := range avMap {
		// skip primary keys
		if field == "TransactionID" || field == "AccountID" {
			continue
		}

		// skip if not allowed to update
		if !config.DBConfig.AllowedUpdateFields[field] {
			continue
		}

		// If it's effectively empty, skip it
		if isEmpty(av) {
			continue
		}

		// Now we convert 'av' back to a standard Go type, e.g. string or float64
		var plainVal interface{}
		if err := attributevalue.Unmarshal(av, &plainVal); err != nil {
			return nil, fmt.Errorf("unmarshal error for field [%s]: %w", field, err)
		}
		// Now plainVal is a normal string, float64, etc.
		updateMap[field] = plainVal
	}

	if len(updateMap) == 0 {
		return nil, errors.New("no fields to update")
	}

	return updateMap, nil
}

// isEmpty checks if a DynamoDB attribute value is considered empty.
func isEmpty(attr types.AttributeValue) bool {
	switch v := attr.(type) {
	case *types.AttributeValueMemberS: // String
		return v.Value == ""
	case *types.AttributeValueMemberN: // Number
		// "0" or "" means no real update value, unless you do want to allow setting zero.
		return v.Value == "0" || v.Value == ""
	case *types.AttributeValueMemberBOOL:
		// If false is considered "empty" in your context, skip it.
		return !v.Value
	case *types.AttributeValueMemberNULL:
		return v.Value
	default:
		return false
	}
}

func (txn *Transaction) ToDynamoDBAttributeValueMap() map[string]events.DynamoDBAttributeValue {
	val := reflect.ValueOf(*txn)
	typ := reflect.TypeOf(*txn)

	avMap := make(map[string]events.DynamoDBAttributeValue)

	if val.Kind() == reflect.Struct {
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldName := typ.Field(i).Name

			switch field.Kind() {
			case reflect.String:
				avMap[fieldName] = events.NewStringAttribute(field.String())
			case reflect.Float64, reflect.Int:
				avMap[fieldName] = events.NewNumberAttribute(fmt.Sprintf("%v", field.Interface()))
			default:
				avMap[fieldName] = events.NewNullAttribute()
			}
		}
	}

	return avMap
}

// Get subject, message for an email fraud alert
func (txn *Transaction) GetFraudEmailContent() (string, string) {
	return "Suspicious Activity on Your Credit Card", fmt.Sprintf(
		`"CAPITAL ONE: We detected a suspicious transaction on your card ending in 1234 for $%.2f at StoreXYZ
on Jan 31 at 3:15 PM. If this was you, reply YES. If not, reply NO or call us at 1-800-XXX-XXXX immediately.
Do not share your account details with anyone."`, txn.TransactionAmount)
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
