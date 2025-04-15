package main

import (
	"context"
	"log"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/lambda"
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

	topicName := config.SNSMessengerConfig.TopicName
	topicArn, err := messaging.CreateTopic(snsClient, topicName)
	if err != nil {
		log.Fatalf("Failed to create SNS topic: %s\n", err)
	}

	snsMessenger := messaging.NewGfSNSMessenger(snsClient, topicName, topicArn)
	eventDispatcher := events.NewGfEventDispatcher(snsMessenger)
	fraudService := services.NewFraudService(eventDispatcher)
	fraudRetryHandler := handlers.NewFraudRetryHandler(fraudService)

	lambda.Start(fraudRetryHandler.ProcessDLQFraudEvent)
}
