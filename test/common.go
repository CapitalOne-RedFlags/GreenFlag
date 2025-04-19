package test

import (
	"encoding/json"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

func GetTestTransaction(email string) models.Transaction {
	now := time.Now()
	eventID := uuid.New().String()

	return models.Transaction{
		TransactionID:       uuid.New().String(),
		AccountID:           "TEST-" + uuid.New().String(),
		TransactionAmount:   100.50,
		TransactionDate:     now.Format(time.RFC3339),
		TransactionType:     "PURCHASE",
		EventID:             eventID,
		EventLabel:          "TRANSACTION",
		EventTimestamp:      now.Format(time.RFC3339),
		LabelTimestamp:      now.Format(time.RFC3339),
		EntityID:            "ENTITY-" + uuid.New().String(),
		EntityType:          "CUSTOMER",
		Location:            "New York",
		IPAddress:           "192.168.1.1",
		TransactionDuration: 120,
		AccountBalance:      5000.00,
		PhoneNumber:         "+12025550179",
		Email:               email,
		EmailAddress:        email,
		TransactionStatus:   "PENDING",
	}
}

func getDynamoDBEventRecord(txn models.Transaction, dbEventType string) events.DynamoDBEventRecord {
	return events.DynamoDBEventRecord{
		EventID:   uuid.New().String(),
		EventName: dbEventType,
		Change: events.DynamoDBStreamRecord{
			SequenceNumber: uuid.New().String(),
			NewImage:       txn.ToDynamoDBAttributeValueMap(),
		},
	}
}

func getSQSEventRecord(txn models.Transaction) events.SQSMessage {
	res, err := json.Marshal(txn)
	if err != nil {
		panic(err)
	}

	return events.SQSMessage{
		MessageId: uuid.New().String(),
		Body:      string(res),
	}
}
