package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

func TransactionService(ctx context.Context, transactions []models.Transaction, repository *db.TransactionRepository) {
	var wg sync.WaitGroup

	for _, record := range transactions {
		wg.Add(1)
		go func(result models.Transaction) {
			defer wg.Done()
			_, _, err := repository.SaveTransaction(ctx, &result)
			if err != nil {
				//DO something
				fmt.Printf("Error saving transaction in transatin service\n%s", err)
			}
		}(record)
	}
	wg.Wait()

}
