package main

import (
	"context"
	"fmt"
	"log"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {
	ctx := context.Background()
	config.InitializeConfig()

	awsConf, err := config.LoadAWSConfig(ctx)
	if err != nil {
		fmt.Printf("Error loading AWS config in lambda initialization\n%s", err)
	}

	tableName := config.DBConfig.TableName
	dbClient := db.NewDynamoDBClient(dynamodb.NewFromConfig(awsConf.Config), tableName)
	repository := db.NewTransactionRepository(dbClient)
	snsClient := sns.NewFromConfig(awsConf.Config)

	topicName := config.SNSMessengerConfig.TopicName
	twilioUsername := config.SNSMessengerConfig.TwilioUsername
	twiilioPassword := config.SNSMessengerConfig.TwilioPassword
	topicArn, err := messaging.CreateTopic(snsClient, topicName)
	if err != nil {
		log.Fatalf("Failed to create SNS topic: %s\n", err)
	}

	snsMessenger := messaging.NewGfSNSMessenger(snsClient, topicName, topicArn, twilioUsername, twiilioPassword)
	dispathcer := events.NewGfEventDispatcher(snsMessenger)
	responseService := services.NewGfResponseService(dispathcer, repository)
	responseHandler := handlers.NewResponseHandler(responseService)

	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %s\n", err)
	}

	lambda.Start(responseHandler.ProcessResponseEvent)

}
