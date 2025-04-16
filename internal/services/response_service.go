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
	UpdateTransaction(ctx context.Context, messages []models.TwilioMessage) error
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

func (rs *GfResponseService) UpdateTransaction(ctx context.Context, messages []models.TwilioMessage) error {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(messages))

	for _, msg := range messages {
		wg.Add(1)
		go func(msg models.TwilioMessage) {
			defer wg.Done()
			if msg.ParseUserResponse() == "NO" {
				count, err := rs.TransactionRepo.UpdateFraudTransaction(ctx, msg.From, true, "POTENTIAL_FRAUD")
				if err != nil {
					fmt.Printf("Error updating fraud transaction: %s", err)
					errorResults <- err
				}
				if count == 0 {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseUnknown)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						errorResults <- err
					}
				} else {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseFraudConfirmed)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						errorResults <- err
					}
				}
			} else if msg.ParseUserResponse() == "YES" {
				count, err := rs.TransactionRepo.UpdateFraudTransaction(ctx, msg.From, false, "POTENTIAL_FRAUD")
				if err != nil {
					fmt.Printf("Error updating fraud transaction: %s", err)
					errorResults <- err
				}
				if count == 0 {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseUnknown)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						errorResults <- err
					}
				} else {
					err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseFraudRejected)
					if err != nil {
						fmt.Printf("Error dispatching fraud event: %s", err)
						errorResults <- err
					}
				}

			} else {
				err := rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, ResponseInvalidResponse)
				if err != nil {
					fmt.Printf("Error dispatching fraud event: %s", err)
					errorResults <- err
				}
			}
		}(msg)
	}
	wg.Wait()
	close(errorResults)

	return middleware.MergeErrors(errorResults)
}
