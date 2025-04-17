package main

import (
	"context"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
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

	service := services.NewTransactionService(repository)
	handler := handlers.NewTransactionProcessingHandler(service)

	// Initialize OpenTelemetry
	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		fmt.Printf("Error initializing OpenTelemetry tracer provider\n%s", err)
	}

	defer func(ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			fmt.Printf("Error shutting down OpenTelemetry tracer provider: %v\n", err)
		}
	}(ctx)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	// Start Lambda with OpenTelemetry instrumentation
	lambda.Start(otellambda.InstrumentHandler(handler.TransactionProcessingHandler, xrayconfig.WithRecommendedOptions(tp)...))
}
