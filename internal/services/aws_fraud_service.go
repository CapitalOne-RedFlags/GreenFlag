package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/events"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector/types"
)

type AWSFraudService struct {
	EventDispatcher events.EventDispatcher
	TransactionRepo db.TransactionRepository
	Client          *frauddetector.Client
}

func NewAWSFraudService(dispatcher events.EventDispatcher, repo db.TransactionRepository) (*AWSFraudService, error) {
	// Load default AWS credentials and region config
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Fraud Detector client
	client := frauddetector.NewFromConfig(cfg)

	return &AWSFraudService{
		EventDispatcher: dispatcher,
		TransactionRepo: repo,
		Client:          client,
	}, nil
}

func (fs *AWSFraudService) PredictFraud(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, error) {
	var fraudulentTransactions []models.Transaction
	var wg sync.WaitGroup
	errorResults := make(chan error, len(transactions))
	fraudResults := make(chan models.Transaction, len(transactions))

	for _, txn := range transactions {
		wg.Add(1)
		go func(txn models.Transaction) {
			defer wg.Done()

			// Format current time in ISO 8601
			eventTime := time.Now().UTC().Format("2006-01-02T15:04:05Z")

			// Build GetEventPredictionInput
			input := &frauddetector.GetEventPredictionInput{
				DetectorId:        aws.String("transaction_detector"),
				DetectorVersionId: aws.String("1"),
				EventId:           aws.String(txn.TransactionID),
				EventTypeName:     aws.String("transaction_event"),
				EventTimestamp:    aws.String(eventTime),
				Entities: []types.Entity{
					{
						EntityType: aws.String("customer"),
						EntityId:   aws.String(txn.AccountID),
					},
				},
				EventVariables: map[string]string{
					"ip_address":           txn.IPAddress,
					"transaction_amount":   fmt.Sprintf("%.2f", txn.TransactionAmount),
					"email_address":        txn.Email,
					"transaction_id":       txn.TransactionID,
					"account_id":           txn.AccountID,
					"transaction_date":     txn.TransactionDate,
					"transaction_type":     txn.TransactionType,
					"location":             txn.Location,
					"transaction_duration": fmt.Sprintf("%d", txn.TransactionDuration),
					"account_balance":      fmt.Sprintf("%.2f", txn.AccountBalance),
					"phone_number":         txn.PhoneNumber,
					"device_id":            txn.DeviceID,
					"merchant_id":          txn.MerchantID,
					"channel":              txn.Channel,
					"customer_age":         fmt.Sprintf("%d", txn.CustomerAge),
					"customer_occupation":  txn.CustomerOccupation,
					"login_attempts":       fmt.Sprintf("%d", txn.LoginAttempts),
				},
			}

			// Call the API
			result, err := fs.Client.GetEventPrediction(ctx, input)
			if err != nil {
				errorResults <- fmt.Errorf("prediction error for transaction %s: %w", txn.TransactionID, err)
				return
			}

			// Check if any model predicts fraud
			isFraud := false
			for _, score := range result.ModelScores {
				if fraudScore, exists := score.Scores["fraud_score"]; exists && fraudScore > 0.5 {
					isFraud = true
					break
				}
			}

			if isFraud {
				// Update transaction status
				txn.TransactionStatus = "POTENTIAL_FRAUD"
				_, err := fs.TransactionRepo.UpdateTransaction(ctx, txn.AccountID, txn.TransactionID, &txn)
				if err != nil {
					errorResults <- fmt.Errorf("failed to update transaction status: %w", err)
					return
				}

				// Dispatch fraud alert
				err = fs.EventDispatcher.DispatchFraudAlertEvent(txn)
				if err != nil {
					errorResults <- fmt.Errorf("failed to dispatch fraud alert: %w", err)
					return
				}

				fraudResults <- txn
			} else {
				// Update transaction status to approved
				txn.TransactionStatus = "APPROVED"
				_, err := fs.TransactionRepo.UpdateTransaction(ctx, txn.AccountID, txn.TransactionID, &txn)
				if err != nil {
					errorResults <- fmt.Errorf("failed to update transaction status: %w", err)
					return
				}
			}
		}(txn)
	}

	wg.Wait()
	close(errorResults)
	close(fraudResults)

	// Collect fraud transactions
	for txn := range fraudResults {
		fraudulentTransactions = append(fraudulentTransactions, txn)
	}

	return fraudulentTransactions, middleware.MergeErrors(errorResults)
}
