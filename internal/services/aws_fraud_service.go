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

type AWSFraudService struct {
	EventDispatcher events.EventDispatcher
	TransactionRepo db.TransactionRepository
	FraudDetector   fraud_detection.FraudDetector
}

func NewAWSFraudService(dispatcher events.EventDispatcher, repo db.TransactionRepository, detector fraud_detection.FraudDetector) *AWSFraudService {
	return &AWSFraudService{
		EventDispatcher: dispatcher,
		TransactionRepo: repo,
		FraudDetector:   detector,
	}
}

func (fs *AWSFraudService) PredictFraud(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, []models.Transaction, error) {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	fraudResults := make(chan models.Transaction, len(transactions))
	approvedResults := make(chan models.Transaction, len(transactions))

	for _, txn := range transactions {
		wg.Add(1)
		go func(txn models.Transaction) {
			defer wg.Done()

			// Use the fraud detector to predict fraud
			isFraud, err := fs.FraudDetector.PredictFraud(ctx, txn)
			if err != nil {
				errorResults <- fmt.Errorf("fraud prediction failed for transaction %s: %w", txn.TransactionID, err)
				return
			}

			if isFraud {
				// Mark as potential fraud
				txn.TransactionStatus = "POTENTIAL_FRAUD"
				// Dispatch fraud alert event
				if err := fs.EventDispatcher.DispatchFraudAlertEvent(txn); err != nil {
					errorResults <- fmt.Errorf("failed to dispatch fraud alert for transaction %s: %w", txn.TransactionID, err)
					return
				}
				fraudResults <- txn
			} else {
				// Mark as approved
				txn.TransactionStatus = "APPROVED"
				approvedResults <- txn
			}

			// Update transaction in database
			_, err = fs.TransactionRepo.UpdateTransaction(ctx, txn.AccountID, txn.TransactionID, &txn)
			if err != nil {
				errorResults <- fmt.Errorf("failed to update transaction %s: %w", txn.TransactionID, err)
				return
			}
		}(txn)
	}

	wg.Wait()
	close(errorResults)
	close(fraudResults)
	close(approvedResults)

	// Collect fraud transactions
	var fraudulentTransactions []models.Transaction
	for txn := range fraudResults {
		fraudulentTransactions = append(fraudulentTransactions, txn)
	}

	// Collect approved transactions
	var approvedTransactions []models.Transaction
	for txn := range approvedResults {
		approvedTransactions = append(approvedTransactions, txn)
	}

	return fraudulentTransactions, approvedTransactions, middleware.MergeErrors(errorResults)
}
