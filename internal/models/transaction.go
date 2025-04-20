package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator"
)

// Transaction represents a record in DynamoDB.
type Transaction struct {
	TransactionID           string  `json:"transaction_id" dynamodbav:"TransactionID" validate:"required"`
	AccountID               string  `json:"account_id" dynamodbav:"AccountID" validate:"required"`
	TransactionAmount       float64 `json:"transaction_amount" dynamodbav:"TransactionAmount" validate:"gte=0"`
	TransactionDate         string  `json:"transaction_date" dynamodbav:"TransactionDate" validate:"required"`
	TransactionType         string  `json:"transaction_type" dynamodbav:"TransactionType" validate:"required"`
	Location                string  `json:"location" dynamodbav:"Location"`
	DeviceID                string  `json:"deviceId" dynamodbav:"DeviceID"`
	IPAddress               string  `json:"ip_address" dynamodbav:"IPAddress" validate:"omitempty,ip"`
	MerchantID              string  `json:"merchantId" dynamodbav:"MerchantID"`
	Channel                 string  `json:"channel" dynamodbav:"Channel"`
	CustomerAge             int     `json:"customerAge" dynamodbav:"CustomerAge" validate:"gte=18"`
	CustomerOccupation      string  `json:"customerOccupation" dynamodbav:"CustomerOccupation"`
	TransactionDuration     int     `json:"transaction_duration" dynamodbav:"TransactionDuration" validate:"gte=0"`
	LoginAttempts           int     `json:"loginAttempts" dynamodbav:"LoginAttempts"`
	AccountBalance          float64 `json:"account_balance" dynamodbav:"AccountBalance" validate:"gte=0"`
	PreviousTransactionDate string  `json:"previousTransactionDate" dynamodbav:"PreviousTransactionDate"`
	PhoneNumber             string  `json:"phone_number" dynamodbav:"PhoneNumber" validate:"required,e164"`
	Email                   string  `json:"email" dynamodbav:"Email" validate:"required,email"`
	TransactionStatus       string  `json:"transaction_status" dynamodbav:"TransactionStatus" validate:"required,oneof=PENDING COMPLETED FAILED FRAUD_DETECTED"`
	EventID                 string  `json:"event_id" dynamodbav:"EventID" validate:"required"`
	EventLabel              string  `json:"event_label" dynamodbav:"EventLabel"`
	EventTimestamp          string  `json:"event_timestamp" dynamodbav:"EventTimestamp" validate:"required"`
	LabelTimestamp          string  `json:"label_timestamp" dynamodbav:"LabelTimestamp"`
	EntityID                string  `json:"entity_id" dynamodbav:"EntityID"`
	EntityType              string  `json:"entity_type" dynamodbav:"EntityType"`
	EmailAddress            string  `json:"email_address" dynamodbav:"EmailAddress" validate:"required,email"`
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

func last4(accountId string) string {
	if len(accountId) >= 4 {
		return accountId[len(accountId)-4:]
	}
	return accountId
}

func formatDateTime(dt string) string {
	t, err := time.Parse(time.RFC3339, dt)
	if err != nil {
		return dt // fallback to raw string
	}
	return t.Format("Jan 2 at 3:04 PM")
}

// Get subject, message for an email fraud alert
func (txn *Transaction) GetFraudEmailContent() (string, string) {
	return "Suspicious Activity on Your Card", fmt.Sprintf("CAPITAL ONE: We detected a suspicious transaction on your card ending in 1234 for $%.2f at %s on %s. If this was you, reply YES. If not, reply NO or call us immediately.", txn.TransactionAmount,
		txn.MerchantID,
		formatDateTime(txn.TransactionDate),
	)

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

func parseTransaction(record []string, colMap map[string]int) (Transaction, error) {
	// Parse numeric fields
	transactionDuration, _ := strconv.Atoi(record[colMap["transaction_duration"]])
	accountBalance, _ := strconv.ParseFloat(record[colMap["account_balance"]], 64)
	transactionAmount, _ := strconv.ParseFloat(record[colMap["transaction_amount"]], 64)

	// Format phone number - add "+" prefix if not present
	phoneNumber := record[colMap["phone_number"]]
	if phoneNumber != "" && !strings.HasPrefix(phoneNumber, "+") {
		phoneNumber = "+" + phoneNumber
	}

	// Create transaction with new fields
	transaction := Transaction{
		// Primary Keys
		TransactionID: record[colMap["transaction_id"]],
		AccountID:     record[colMap["account_id"]],

		// Transaction Details
		TransactionAmount: transactionAmount,
		TransactionDate:   record[colMap["transaction_date"]],
		TransactionType:   record[colMap["transaction_type"]],

		// Event Information
		EventID:        record[colMap["EVENT_ID"]],
		EventLabel:     record[colMap["EVENT_LABEL"]],
		EventTimestamp: record[colMap["EVENT_TIMESTAMP"]],
		LabelTimestamp: record[colMap["LABEL_TIMESTAMP"]],

		// Entity Information
		EntityID:   record[colMap["ENTITY_ID"]],
		EntityType: record[colMap["ENTITY_TYPE"]],

		// Location and Network Info
		Location:  record[colMap["location"]],
		IPAddress: record[colMap["ip_address"]],

		// Transaction Metadata
		TransactionDuration: transactionDuration,

		// Financial Information
		AccountBalance: accountBalance,

		// Contact Information
		PhoneNumber:  phoneNumber,
		Email:        record[colMap["email"]],
		EmailAddress: record[colMap["email_address"]],

		// Default status for new transactions
		TransactionStatus: "PENDING",
	}

	return transaction, nil
}
