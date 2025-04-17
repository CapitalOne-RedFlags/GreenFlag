package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
)

// GetFraudScore evaluates a transaction for fraud.
func GetFraudScore(client *frauddetector.Client) {
	response, err := client.GetPrediction(context.TODO(), &frauddetector.GetPredictionInput{
		DetectorId:   aws.String("transaction_detector"),
		EventId:      aws.String("event123"),
		EventTypeName: aws.String("transaction_event"),
		Entities: []frauddetector.Entity{
			{Name: aws.String("customer"), EntityId: aws.String("cust123")},
		},
		EventVariables: map[string]string{
			"ip_address":         "192.168.1.1",
			"transaction_amount": "500.00",
		},
	})
	if err != nil {
		log.Fatalf("Failed to get fraud score: %v", err)
	}
	fmt.Println("Fraud Score:", response.RuleResults)
}
