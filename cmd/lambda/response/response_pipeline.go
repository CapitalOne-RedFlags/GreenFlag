package responseMain

import (
	"context"
	"fmt"
	"log"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	fmt.Printf("This is a test to see if response is running")
	ctx := context.Background()
	config.InitializeConfig()

	awsConf, err := config.LoadAWSConfig(ctx)
	if err != nil {
		fmt.Printf("Error loading AWS config in lambda initialization\n%s", err)
	}

	tableName := config.DBConfig.TableName
	dbClient := db.NewDynamoDBClient(dynamodb.NewFromConfig(awsConf.Config), tableName)
	repository := db.NewTransactionRepository(dbClient)

	responseService := services.NewGfResponseService()
	responseHandler := handlers.NewResponseHandler(responseService, repository)

	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %s\n", err)
	}

	lambda.Start(responseHandler.ProcessResponseEvent)

}
