package services

import (
	"context"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type TransactionService struct {
	repository db.TransactionRepository
}

func NewTransactionService(repository db.TransactionRepository) *TransactionService {
	return &TransactionService{
		repository: repository,
	}
}

func (ts *TransactionService) TransactionService(ctx context.Context, transactions []models.Transaction) error {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	for _, record := range transactions {
		wg.Add(1)
		go func(result models.Transaction) {
			defer wg.Done()
			_, _, err := ts.repository.SaveTransaction(ctx, &result)
			if err != nil {
				errorResults <- err
			}
		}(record)
	}
	wg.Wait()
	close(errorResults)
	return middleware.MergeErrors(errorResults)
}

func (ts *TransactionService) SaveTransaction(ctx context.Context, txn models.Transaction, wg *sync.WaitGroup) error {
	errorResult := make(chan error)
	wg.Add(1)
	go func(result models.Transaction) {
		defer wg.Done()
		_, _, err := ts.repository.SaveTransaction(ctx, &txn)
		if err != nil {
			errorResult <- err
		}
	}(txn)
	wg.Wait()

	return <-errorResult
}
