package main

import (
	"context"
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
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
)

func main() {
	config.InitializeConfig()

	context := context.Background()

	awsConfig, err := config.LoadAWSConfig(context)
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %s\n", err)
	}
	tableName := config.DBConfig.TableName
	dbClient := db.NewDynamoDBClient(dynamodb.NewFromConfig(awsConfig.Config), tableName)
	repository := db.NewTransactionRepository(dbClient)
	snsClient := sns.NewFromConfig(awsConfig.Config)

	topicName := config.SNSMessengerConfig.TopicName
	twilioUsername := config.SNSMessengerConfig.TwilioUsername
	twiilioPassword := config.SNSMessengerConfig.TwilioPassword
	topicArn, err := messaging.CreateTopic(snsClient, topicName)
	if err != nil {
		log.Fatalf("Failed to create SNS topic: %s\n", err)
	}
	// Initialize OpenTelemetry
	tp, err := xrayconfig.NewTracerProvider(context)
	if err != nil {
		log.Fatalf("Error initializing OpenTelemetry tracer provider: %s\n", err)
	}

	defer func() {
		err := tp.Shutdown(context)
		if err != nil {
			log.Fatalf("Error shutting down OpenTelemetry tracer provider: %s\n", err)
		}
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	snsMessenger := messaging.NewGfSNSMessenger(snsClient, topicName, topicArn, twilioUsername, twiilioPassword)
	eventDispatcher := events.NewGfEventDispatcher(snsMessenger)
	fraudService, err := services.NewAWSFraudService(eventDispatcher, repository)
	if err != nil {
		log.Fatalf("Failed to initialize AWS Fraud Detector service: %s\n", err)
	}
	fraudHandler := handlers.NewFraudHandler(fraudService)

	lambda.Start(otellambda.InstrumentHandler(fraudHandler.ProcessFraudEvent, xrayconfig.WithRecommendedOptions(tp)...))
}
