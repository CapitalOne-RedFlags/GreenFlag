package fraudMain

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

	snsMessenger := messaging.NewGfSNSMessenger(snsClient)
	eventDispatcher := events.NewGfEventDispatcher(snsMessenger)
	fraudService := services.NewFraudService(eventDispatcher)
	fraudHandler := handlers.NewFraudHandler(fraudService)

	lambda.Start(fraudHandler.ProcessFraudEvent)
}
