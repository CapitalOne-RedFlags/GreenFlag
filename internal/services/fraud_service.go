package services

import (
	"sync"

	"slices"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type FraudService interface {
	PredictFraud(transactions []models.Transaction) error
}

type GfFraudService struct {
	EventDispatcher events.EventDispatcher
}

func NewFraudService(e events.EventDispatcher) *GfFraudService {
	return &GfFraudService{
		EventDispatcher: e,
	}
}

func (fs *GfFraudService) PredictFraud(transactions []models.Transaction) error {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))

	for _, txn := range transactions {
		wg.Add(1)
		go func(txn models.Transaction) {
			defer wg.Done()
			isFraud, err := predictFraud(txn)
			if err != nil {
				errorResults <- err
			}

			if isFraud {
				err := fs.EventDispatcher.DispatchFraudAlertEvent(txn)
				if err != nil {
					errorResults <- err
				}
			}
		}(txn)
	}
	wg.Wait()
	close(errorResults)

	return middleware.MergeErrors(errorResults)
}

// Placeholder for fraud prediction, to be replaced with prediction algorithm
func predictFraud(transaction models.Transaction) (bool, error) {
	return slices.Contains([]string{"rshart@wisc.edu", "jpconnell4@wisc.eud"}, transaction.Email), nil
}
