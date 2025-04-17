package services

import (
	"context"
	"fmt"
	"sync"

	"slices"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type FraudService interface {
	PredictFraud(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, []models.Transaction, error)
}

type GfFraudService struct {
	EventDispatcher events.EventDispatcher
	TransactionRepo db.TransactionRepository
}

func NewFraudService(dispatcher events.EventDispatcher, repo db.TransactionRepository) *GfFraudService {
	return &GfFraudService{
		EventDispatcher: dispatcher,
		TransactionRepo: repo,
	}
}

func (fs *GfFraudService) PredictFraud(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, []models.Transaction, error) {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	failedTransactions := make(chan models.Transaction, len(transactions))
	fraudulentTransactions := make(chan models.Transaction, len(transactions))

	for _, txn := range transactions {
		wg.Add(1)
		go func(txn models.Transaction) {
			defer wg.Done()
			isFraud, err := predictFraud(txn)
			if err != nil {
				errorResults <- err
				failedTransactions <- txn
				return
			}

			if isFraud {
				fraudulentTransactions <- txn
				err := fs.EventDispatcher.DispatchFraudAlertEvent(txn)
				if err != nil {
					wrappedErr := fmt.Errorf("fraud prediction failed for transaction %s (account: %s, amount: %.2f, merchant: %s, email: %s): %w",
						txn.TransactionID,
						txn.AccountID,
						txn.TransactionAmount,
						txn.MerchantID,
						txn.Email,
						err)
					errorResults <- wrappedErr
					failedTransactions <- txn
					return
				} else {
					txn.TransactionStatus = "POTENTIAL_FRAUD"
					_, err := fs.TransactionRepo.UpdateTransaction(
						ctx,
						txn.AccountID,
						txn.TransactionID,
						&txn,
					)
					if err != nil {
						wrappedErr := fmt.Errorf("fraud prediction failed for transaction %s (account: %s, amount: %.2f, merchant: %s, email: %s): %w",
							txn.TransactionID,
							txn.AccountID,
							txn.TransactionAmount,
							txn.MerchantID,
							txn.Email,
							err)
						errorResults <- wrappedErr
						failedTransactions <- txn
						return
					}
				}
			} else {
				txn.TransactionStatus = "APPROVED"
				_, err := fs.TransactionRepo.UpdateTransaction(
					ctx,
					txn.AccountID,
					txn.TransactionID,
					&txn,
				)
				if err != nil {
					wrappedErr := fmt.Errorf("fraud prediction failed for transaction %s (account: %s, amount: %.2f, merchant: %s, email: %s): %w",
						txn.TransactionID,
						txn.AccountID,
						txn.TransactionAmount,
						txn.MerchantID,
						txn.Email,
						err)
					errorResults <- wrappedErr
					failedTransactions <- txn
					return
				}
			}
		}(txn)
	}
	wg.Wait()
	close(errorResults)
	close(failedTransactions)
	close(fraudulentTransactions)

	return channelToSlice(fraudulentTransactions), channelToSlice(failedTransactions), middleware.MergeErrors(errorResults)
}

// Placeholder for fraud prediction, to be replaced with prediction algorithm
func predictFraud(transaction models.Transaction) (bool, error) {
	return slices.Contains([]string{"rshart@wisc.edu", "jpoconnell4@wisc.edu", "c1redflagstest@gmail.com", "wlee298@wisc.edu", "donglaiduann@gmail.com"}, transaction.Email), nil
}
