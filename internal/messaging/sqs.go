package messaging

import (
	"context"
	"encoding/json"
	"log"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSHandler struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSHandler(client *sqs.Client, queueURL string) *SQSHandler {
	return &SQSHandler{
		client:   client,
		queueURL: queueURL,
	}
}

// SendTransaction sends a transaction to SQS
func (h *SQSHandler) SendTransaction(ctx context.Context, transaction *models.Transaction) error {
	jsonData, err := json.Marshal(transaction)
	if err != nil {
		return err
	}

	// Log the JSON data being sent to SQS
	log.Printf("Sending transaction to SQS: %s", string(jsonData))

	_, err = h.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(h.queueURL),
		MessageBody: aws.String(string(jsonData)),
	})
	return err
}

// ReceiveTransactions receives messages from SQS
func (h *SQSHandler) ReceiveTransactions(ctx context.Context) ([]*models.Transaction, error) {
	output, err := h.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(h.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20, // Long polling
	})
	if err != nil {
		return nil, err
	}

	var transactions []*models.Transaction
	for _, msg := range output.Messages {
		var transaction models.Transaction
		if err := json.Unmarshal([]byte(*msg.Body), &transaction); err != nil {
			continue
		}
		transactions = append(transactions, &transaction)

		// Delete the message after processing
		_, err = h.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(h.queueURL),
			ReceiptHandle: msg.ReceiptHandle,
		})
		if err != nil {
			// Log error but continue processing
			continue
		}
	}

	return transactions, nil
}
