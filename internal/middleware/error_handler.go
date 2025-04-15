package middleware

import (
	"context"
	"errors"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func MergeErrors(errCh <-chan error) error {
	result := []error{}
	for err := range errCh {
		result = append(result, err)
	}

	return errors.Join(result...)
}

func WithManualDLQHandling(txn models.Transaction, dlqUrl string, serviceFunc func(models.Transaction) error) error {
	err := serviceFunc(txn)
	if err == nil {
		return nil
	}
 
	if config.HandlerConfig.IsRetry {
		sqsClient := sqs.Client{}
		sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
			QueueUrl: aws.String(dlqUrl),
		})
	}
}
