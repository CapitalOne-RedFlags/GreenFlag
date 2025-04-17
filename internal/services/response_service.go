package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type ResponseService interface {
	RsUpdateTransaction(ctx context.Context, messages []models.TwilioMessage) ([]models.TwilioMessage, error)
}

type GfResponseService struct {
	EventDispatcher events.EventDispatcher
	TransactionRepo db.TransactionRepository
}

const (
	ResponseFraudConfirmed  = "Thank you for your response. We have canceled this transaction. Your balance will be updated accordingly."
	ResponseFraudRejected   = "Thank you for your response. We have updated this transaction status to valid. Your balance will be updated accordingly."
	ResponseInvalidResponse = "If texted about fraud, please reply YES if this was you or NO if it was not. Otherwise do not text this number."
	ResponseUnknown         = "Please do not text this number unless prompted"
)

func NewGfResponseService(dispatcher events.EventDispatcher, repo db.TransactionRepository) *GfResponseService {
	return &GfResponseService{
		EventDispatcher: dispatcher,
		TransactionRepo: repo,
	}
}

func (rs *GfResponseService) RsUpdateTransaction(ctx context.Context, messages []models.TwilioMessage) ([]models.TwilioMessage, error) {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(messages))
	failedMessages := make(chan models.TwilioMessage, len(messages))
	for _, msg := range messages {
		wg.Add(1)
		go func(msg models.TwilioMessage) {
			defer wg.Done()
			if msg.ParseUserResponse() == "NO" {
				count, err := rs.TransactionRepo.UpdateFraudTransaction(ctx, msg.From, true, "POTENTIAL_FRAUD")
				if err != nil {
					fmt.Printf("Error updating fraud transaction: %s", err)
					failedMessages <- msg
					errorResults <- err
				}
				if count == 0 {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseUnknown)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						failedMessages <- msg
						errorResults <- err
					}
				} else {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseFraudConfirmed)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						failedMessages <- msg
						errorResults <- err
					}
				}
			} else if msg.ParseUserResponse() == "YES" {
				count, err := rs.TransactionRepo.UpdateFraudTransaction(ctx, msg.From, false, "POTENTIAL_FRAUD")
				if err != nil {
					fmt.Printf("Error updating fraud transaction: %s", err)
					failedMessages <- msg
					errorResults <- err
				}
				if count == 0 {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseUnknown)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						failedMessages <- msg
						errorResults <- err
					}
				} else {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseFraudRejected)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						failedMessages <- msg
						errorResults <- err
					}
				}

			} else {
				err := rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseInvalidResponse)
				if err != nil {
					fmt.Printf("Error dispatching fraud event: %s", err)
					failedMessages <- msg
					errorResults <- err
				}
			}
		}(msg)
	}
	wg.Wait()
	close(errorResults)
	close(failedMessages)

	return channelToSlice(failedMessages), middleware.MergeErrors(errorResults)
}
func channelToSlice[T any](ch <-chan T) []T {
	var result []T
	for val := range ch {
		result = append(result, val)
	}
	return result
}
