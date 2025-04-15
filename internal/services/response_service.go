package services

import (
	"context"
	"fmt"
	"strings"
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
			if strings.ToUpper(strings.TrimSpace(msg.Body)) == "NO" {
				err := rs.TransactionRepo.UpdateFraudTransaction(ctx, msg.From, true)
				if err != nil {
					fmt.Printf("Error updating fraud transaction: %s", err)
					errorResults <- err
				}
				body := "Thank you for your response. We have canceled this transaction. Your balance will be updated accordingly"
				err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, body)
				if err != nil {
					fmt.Printf("Error dispatching fraud event: %s", err)
					errorResults <- err
				}
			} else if strings.ToUpper(strings.TrimSpace(msg.Body)) == "YES" {
				err := rs.TransactionRepo.UpdateFraudTransaction(ctx, msg.From, false)
				if err != nil {
					fmt.Printf("Error updating fraud transaction: %s", err)
					errorResults <- err
				}
				body := "Thank you for your response. We have updated this trascation status to valid. Your balance will be updated accordingly"
				err = rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, body)
				if err != nil {
					fmt.Printf("Error dispatching fraud event: %s", err)
					errorResults <- err
				}
			} else {
				body := "Please reply YES if this was you or NO if it was not"
				err := rs.EventDispatcher.DispatchFraudUpdateEvent(msg.From, body)
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
