package handlers

import (
	"context"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

type ResponseHandler interface {
	ProcessResponseEvent(ctx context.Context, event events.SQSEvent) error
}

type GfResponseHandler struct {
	responseService services.ResponseService
}

func NewResponseHandler(responseService services.ResponseService) *GfResponseHandler {
	return &GfResponseHandler{
		responseService: responseService,
	}
}

func (rh *GfResponseHandler) ProcessResponseEvent(ctx context.Context, event events.SQSEvent) error {
	var messages []models.TwilioMessage
	for _, record := range event.Records {
		result, err := models.UnmarshalResponseSQS(record.Body)
		if err != nil {
			fmt.Printf("error Unmarshalling: %s", err)
			return err
		}
		messages = append(messages, *result)
	}
	_, err := rh.responseService.RsUpdateTransaction(ctx, messages)
	return err
}
