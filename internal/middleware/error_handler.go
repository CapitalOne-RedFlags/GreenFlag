package middleware

import (
	"errors"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
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

func GetBatchResult(input *GetBatchResultInput) (*events.BatchResult, error) {
	var results []events.BatchItemFailure

	for _, txn := range input.FailedTransactions {
		results = append(results, events.BatchItemFailure{
			ItemIdentifier: input.RIDsByTransactionId[txn.TransactionID],
		})
	}

	for _, rid := range input.FailedRIDs {
		results = append(results, events.BatchItemFailure{
			ItemIdentifier: rid,
		})
	}

	return &events.BatchResult{
		BatchItemFailures: results,
	}, errors.Join(input.Errors...)
}
