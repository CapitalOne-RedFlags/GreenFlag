package services

import (
	"context"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

func TransactionService(ctx context.Context, transactions []models.Transaction, repository db.TransactionRepository) error {
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	for _, record := range transactions {
		wg.Add(1)
		go func(result models.Transaction) {
			defer wg.Done()
			_, _, err := repository.SaveTransaction(ctx, &result)
			if err != nil {
				errorResults <- err
			}
		}(record)
	}
	wg.Wait()
	close(errorResults)
	return middleware.MergeErrors(errorResults)
}
