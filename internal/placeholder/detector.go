package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// CreateEntityType creates an entity type in AWS Fraud Detector.
func CreateEntityType(client *frauddetector.Client) {
	_, err := client.PutEntityType(context.TODO(), &frauddetector.PutEntityTypeInput{
		Name:        aws.String("customer"),
		Description: aws.String("Customer entity type"),
	})
	if err != nil {
		log.Fatalf("Failed to create entity type: %v", err)
	}
	fmt.Println("Entity type created.")
}

// CreateEventType creates an event type for fraud detection.
func CreateEventType(client *frauddetector.Client) {
	_, err := client.PutEventType(context.TODO(), &frauddetector.PutEventTypeInput{
		Name: aws.String("transaction_event"),
		EntityTypes: []frauddetector.EntityType{
			{Name: aws.String("customer")},
		},
		EventVariables: []frauddetector.EventVariable{
			{Name: aws.String("ip_address")}, // todo add necessary event variables
			{Name: aws.String("transaction_amount")},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create event type: %v", err)
	}
	fmt.Println("Event type created.")
}

// CreateDetector creates a fraud detector.
func CreateDetector(client *frauddetector.Client) {
	_, err := client.PutDetector(context.TODO(), &frauddetector.PutDetectorInput{
		DetectorId:    aws.String("transaction_detector"),
		EventTypeName: aws.String("transaction_event"),
	})
	if err != nil {
		log.Fatalf("Failed to create detector: %v", err)
	}
	fmt.Println("Detector created.")
}
