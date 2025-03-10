package middleware

import (
	"errors"
)

func MergeErrors(errCh <-chan error) error {
	result := []error{}
	for err := range errCh {
		result = append(result, err)
	}

	return errors.Join(result...)
}
