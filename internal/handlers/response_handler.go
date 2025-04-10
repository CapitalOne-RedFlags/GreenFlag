package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
)

type ResponseHandler interface {
	ProcessResponseEvent(ctx context.Context, event events.SQSEvent) error
}

type GfResponseHandler struct {
	responseService services.ResponseService
	repository      db.TransactionRepository
}

func NewResponseHandler(responseService services.ResponseService, repository db.TransactionRepository) *GfResponseHandler {
	return &GfResponseHandler{
		responseService: responseService,
		repository:      repository,
	}
}

func (rh *GfResponseHandler) ProcessResponseEvent(ctx context.Context, event events.SQSEvent) error {
	var transactions []models.Transaction
	for _, record := range event.Records {

		jsonData, erro := json.Marshal(record.Body)
		if erro != nil {
			fmt.Println("Error marshaling JSON:", erro)
			return erro
		}

		// Convert the byte slice to a string
		jsonString := string(jsonData)

		// Print the JSON string
		fmt.Printf("MESSAGE: %s\n", jsonString)
		result, err := models.UnmarshalSQS(record.Body)
		if err != nil {
			return err
		}
		transactions = append(transactions, *result)
	}

	return nil
}
