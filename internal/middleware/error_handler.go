package middleware

import (
	"errors"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

func MergeErrors(errCh <-chan error) error {
	result := []error{}
	for err := range errCh {
		result = append(result, err)
	}

	return errors.Join(result...)
}

type GetBatchResultInput struct {
	FailedTransactions  []models.Transaction
	RIDsByTransactionId map[string]string
	FailedRIDs          []string
	Errors              []error
}

func GetBatchResult(input *GetBatchResultInput) (*models.BatchResult, error) {
	var results []models.BatchItemFailure

	for _, txn := range input.FailedTransactions {
		results = append(results, models.BatchItemFailure{
			ItemIdentifier: input.RIDsByTransactionId[txn.TransactionID],
		})
	}

	for _, rid := range input.FailedRIDs {
		results = append(results, models.BatchItemFailure{
			ItemIdentifier: rid,
		})
	}

	return &models.BatchResult{
		BatchItemFailures: results,
	}, errors.Join(input.Errors...)
}
