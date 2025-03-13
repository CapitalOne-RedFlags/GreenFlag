package main

import (
	"context"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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

	handlerWithRepo := func(ctx context.Context, event events.SQSEvent) {
		tpErr := handlers.TransactionProcessingHandler(ctx, event, repository)
		if tpErr != nil {
			fmt.Printf("Error initializing transaction processing handler:\n%s", tpErr)
		}
	}
	lambda.Start(handlerWithRepo)
}
