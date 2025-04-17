package services

import (
	"context"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

type TransactionService interface {
	TransactionService(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, error)
}

type GfTransactionService struct {
	repository db.TransactionRepository
}

func NewTransactionService(repository db.TransactionRepository) *GfTransactionService {
	return &GfTransactionService{
		repository: repository,
	}
}

func (ts *GfTransactionService) TransactionService(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, error) {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	failedTransactions := make(chan models.Transaction, len(transactions))

	for _, record := range transactions {
		wg.Add(1)
		go func(txn models.Transaction) {
			defer wg.Done()
			_, _, err := ts.repository.SaveTransaction(ctx, &txn)
			if err != nil {
				errorResults <- err
				failedTransactions <- txn
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
