package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/fraud_detection"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type FraudService interface {
	PredictFraud(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, []models.Transaction, error)
}

type GfFraudService struct {
	EventDispatcher events.EventDispatcher
	TransactionRepo db.TransactionRepository
	fraudDetector   fraud_detection.FraudDetector
}

func NewFraudService(dispatcher events.EventDispatcher, repo db.TransactionRepository, detector fraud_detection.FraudDetector) *GfFraudService {
	return &GfFraudService{
		EventDispatcher: dispatcher,
		TransactionRepo: repo,
		fraudDetector:   detector,
	}
}

func (fs *GfFraudService) predictFraud(ctx context.Context, transaction models.Transaction) (bool, error) {
	return fs.fraudDetector.PredictFraud(ctx, transaction)
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

			// Use the fraud detector to predict fraud
			isFraud, err := fs.predictFraud(ctx, txn)
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
