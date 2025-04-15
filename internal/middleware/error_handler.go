package middleware

import (
	"errors"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
)

func MergeErrors(errCh <-chan error) error {
	result := []error{}
	for err := range errCh {
		result = append(result, err)
	}

	return errors.Join(result...)
}

func MergeBatchItemFailures(ch <-chan events.BatchItemFailure) *events.BatchResult {
	result := []events.BatchItemFailure{}
	for batchItemFailure := range ch {
		result = append(result, batchItemFailure)
	}

	return &events.BatchResult{
		BatchItemFailures: result,
	}
}
