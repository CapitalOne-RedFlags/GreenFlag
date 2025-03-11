package fraudMain

import (
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	eventDispatcher := &events.GfEventDispatcher{}
	fraudService := services.NewFraudService(eventDispatcher)
	fraudHandler := handlers.NewFraudHandler(fraudService)

	lambda.Start(fraudHandler.ProcessFraudEvent)
}
