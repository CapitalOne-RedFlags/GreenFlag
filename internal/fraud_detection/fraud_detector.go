package fraud_detection

import (
	"context"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
)

// FraudDetector interface defines the contract for fraud detection
type FraudDetector interface {
	PredictFraud(ctx context.Context, txn models.Transaction) (bool, error)
}

// GfAWSFraudDetector implements the FraudDetector interface using AWS Fraud Detector
type GfAWSFraudDetector struct {
	client *frauddetector.Client
}

// NewGfAWSFraudDetector creates a new instance of GfAWSFraudDetector
func NewGfAWSFraudDetector(client *frauddetector.Client) *GfAWSFraudDetector {
	return &GfAWSFraudDetector{
		client: client,
	}
}

// PredictFraud implements the FraudDetector interface using AWS Fraud Detector
func (fd *GfAWSFraudDetector) PredictFraud(ctx context.Context, txn models.Transaction) (bool, error) {
	// Only use the specified event variables
	eventVariables := map[string]string{
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
	}

	// Create the fraud detection input
	input := &frauddetector.GetEventPredictionInput{
		DetectorId:        aws.String("transaction_detector"),
		DetectorVersionId: aws.String("1"),
		EventId:           aws.String(txn.TransactionID),
		EventTypeName:     aws.String("transaction_event"),
		EventVariables:    eventVariables,
	}

	// Get fraud prediction
	result, err := fd.client.GetEventPrediction(ctx, input)
	if err != nil {
		return false, fmt.Errorf("failed to get fraud prediction for transaction %s: %w", txn.TransactionID, err)
	}

	// Process the prediction result
	if result.ModelScores != nil && len(result.ModelScores) > 0 {
		for _, score := range result.ModelScores {
			if fraudScore, exists := score.Scores["fraud_score"]; exists && fraudScore > 0.5 {
				return true, nil
			}
		}
	}

	return false, nil
}
