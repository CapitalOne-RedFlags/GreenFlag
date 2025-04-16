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

func (ts *TransactionService) TransactionService(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, error) {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	failedTransactions := make(chan models.Transaction, len(transactions))

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
	close(failedTransactions)

	return channelToSlice(failedTransactions), middleware.MergeErrors(errorResults)
}

func channelToSlice[T any](ch <-chan T) []T {
	var result []T
	for val := range ch {
		result = append(result, val)
	}
	return result
}
