package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
)

// CreateModel registers a new fraud detection model.
func CreateModel(client *frauddetector.Client) {
	modelID := "fraud_model"
	modelType := "ONLINE_FRAUD_INSIGHTS"
	description := "Fraud detection model"

	_, err := client.PutModel(context.TODO(), &frauddetector.PutModelInput{
		ModelId:       aws.String(modelID),
		ModelType:     frauddetector.ModelType(modelType),
		Description:   aws.String(description),
		EventTypeName: aws.String("transaction_event"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}
	fmt.Println("Model created successfully.")
}
