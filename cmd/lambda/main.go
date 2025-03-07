package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"your-project/internal/models"
	"your-project/internal/db"
)

func handleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	// Initialize DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)
	repo := db.NewDynamoRepository(dynamoClient)

	// Process each message from SQS
	for _, message := range sqsEvent.Records {
		// Parse the transaction from the message
		var transaction models.Transaction
		if err := json.Unmarshal([]byte(message.Body), &transaction); err != nil {
			log.Printf("Error unmarshaling transaction: %v", err)
			continue
		}

		// Set additional fields
		transaction.Status = "PENDING"
		transaction.CreatedAt = time.Now()
		transaction.UpdatedAt = time.Now()

		// Store in DynamoDB
		if err := repo.SaveTransaction(ctx, &transaction); err != nil {
			log.Printf("Error saving transaction %s: %v", transaction.TransactionID, err)
			continue
		}

		log.Printf("Successfully processed transaction %s", transaction.TransactionID)
	}

	return nil
}

func main() {
	lambda.Start(handleRequest)
}

package main
