package main

import (
	"context"
	"log"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/fraud_detection"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {
	config.InitializeConfig()

	context := context.Background()

	awsConfig, err := config.LoadAWSConfig(context)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %s\n", err)
	}

	snsClient := sns.NewFromConfig(awsConfig.Config)
	fraudDetectorClient := frauddetector.NewFromConfig(awsConfig.Config)

	tableName := config.DBConfig.TableName
	dbClient := db.NewDynamoDBClient(dynamodb.NewFromConfig(awsConfig.Config), tableName)
	repository := db.NewTransactionRepository(dbClient)

	topicName := config.SNSMessengerConfig.TopicName
	topicArn, err := messaging.CreateTopic(snsClient, topicName)
	if err != nil {
		log.Fatalf("Failed to create SNS topic: %s\n", err)
	}

	twilioUsername := config.SNSMessengerConfig.TwilioUsername
	twiilioPassword := config.SNSMessengerConfig.TwilioPassword
	snsMessenger := messaging.NewGfSNSMessenger(snsClient, topicName, topicArn, twilioUsername, twiilioPassword)
	eventDispatcher := events.NewGfEventDispatcher(snsMessenger)
	fraudDetector := fraud_detection.NewGfAWSFraudDetector(fraudDetectorClient)
	fraudService := services.NewFraudService(eventDispatcher, repository, fraudDetector)
	fraudRetryHandler := handlers.NewFraudRetryHandler(fraudService)

	lambda.Start(fraudRetryHandler.ProcessDLQFraudEvent)
}
