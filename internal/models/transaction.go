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

func UnmarshalSQS(trasaction string) (*Transaction, error) {
	var result Transaction
	err := json.Unmarshal([]byte(trasaction), &result)
	if err != nil {
		//Do SOmething
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
