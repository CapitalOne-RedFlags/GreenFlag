package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// SendFraudEvent sends an event to AWS Fraud Detector.
func SendFraudEvent(client *frauddetector.Client) {
	_, err := client.SendEvent(context.TODO(), &frauddetector.SendEventInput{
		EventId:       aws.String("event123"),
		EventTypeName: aws.String("transaction_event"),
		DetectorId:    aws.String("transaction_detector"),
		Entities: []frauddetector.Entity{
			{Name: aws.String("customer"), EntityId: aws.String("cust123")},
		},
		EventTimestamp: aws.String("2025-03-09T12:00:00Z"),
		EventVariables: map[string]string{
			"ip_address":         "192.168.1.1",
			"transaction_amount": "500.00",
		},
	})
	if err != nil {
		log.Fatalf("Failed to send event: %v", err)
	}
	fmt.Println("Event sent successfully.")
}
