package models

package models

import (
	"errors"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator"
)

type Transaction struct {
	TransactionID           string  `dynamodbav:"TransactionID" validate:"required"`
	AccountID               string  `dynamodbav:"AccountID" validate:"required"`
	TransactionAmount       float64 `dynamodbav:"TransactionAmount" validate:"gte=0"`
	TransactionDate         string  `dynamodbav:"TransactionDate"`
	TransactionType         string  `dynamodbav:"TransactionType"`
	Location                string  `dynamodbav:"Location"`
	DeviceID                string  `dynamodbav:"DeviceID"`
	IPAddress               string  `dynamodbav:"IPAddress"`
	MerchantID              string  `dynamodbav:"MerchantID"`
	Channel                 string  `dynamodbav:"Channel"`
	CustomerAge             int     `dynamodbav:"CustomerAge" validate:"gte=18"`
	CustomerOccupation      string  `dynamodbav:"CustomerOccupation"`
	TransactionDuration     int     `dynamodbav:"TransactionDuration"`
	LoginAttempts           int     `dynamodbav:"LoginAttempts"`
	AccountBalance          float64 `dynamodbav:"AccountBalance"`
	PreviousTransactionDate string  `dynamodbav:"PreviousTransactionDate"`
	PhoneNumber             string  `dynamodbav:"PhoneNumber" validate:"required,e164"`
	Email                   string  `dynamodbav:"Email" validate:"required,email"`
	TransactionStatus       string  `dynamodbav:"TransactionStatus"`
}

// Helper functions

// MarshalDynamoDB marshals a Transaction into a DynamoDB attribute map.
func (t *Transaction) MarshalDynamoDB() (map[string]types.AttributeValue, error) {
	return attributevalue.MarshalMap(t)
}

// UnmarshalDynamoDB unmarshals a DynamoDB attribute map into a Transaction.
func UnmarshalDynamoDB(av map[string]types.AttributeValue) (*Transaction, error) {
	var t Transaction
	err := attributevalue.UnmarshalMap(av, &t)
	return &t, err
}

// ValidateTransaction validates an incoming transaction. Call this before processing the transaction.
func (t *Transaction) ValidateTransaction() error {
	validate := validator.New()
	return validate.Struct(t)
}

// Converts a Transaction struct to a DynamoDB update map.
// It ensures only non-empty fields are included.
func (t *Transaction) TransactionUpdatePayload() (map[string]interface{}, error) {
	updateMap := make(map[string]interface{})

	// Convert struct to a DynamoDB map
	item, err := attributevalue.MarshalMap(t)
	if err != nil {
		return nil, err
	}

	// Remove empty or zero-value fields to prevent overwriting with empty data
	for key, value := range item {
		if !config.DBConfig.AllowedUpdateFields[key] {
			return nil, fmt.Errorf("field[%s] not allowed to update", key)
		} else if !isEmpty(value) {
			updateMap[key] = value
		}

	}

	if len(updateMap) == 0 {
		return nil, errors.New("no fields to update")
	}

	return updateMap, nil
}

// isEmpty checks if a DynamoDB attribute value is empty.
func isEmpty(attr types.AttributeValue) bool {
	switch v := attr.(type) {
	case *types.AttributeValueMemberS: // String
		return v.Value == ""
	case *types.AttributeValueMemberN: // Number
		return v.Value == "0" || v.Value == ""
	case *types.AttributeValueMemberBOOL: // Boolean
		return !v.Value // False means empty in some cases
	case *types.AttributeValueMemberNULL: // Null values
		return v.Value
	default:
		return false
	}
}