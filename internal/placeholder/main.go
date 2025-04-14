package main

import (
	"fmt"
)

func main() {
	fmt.Println("Starting AWS Fraud Detector Setup...")

	client := GetAWSConfig()

	// Step 1: Set up entity types and event types
	CreateEntityType(client)
	CreateEventType(client)

	// Step 2: Create the fraud detector
	CreateDetector(client)

	// Step 3: Upload dataset to S3
	UploadToS3("your-bucket-name", "fraud_data.csv", "fraud_data.csv")

	// Step 4: Create and train the Fraud Detector model
	CreateModel(client)
	TrainModel(client)

	// Step 5: Activate the trained model
	ActivateModel(client)

	// Step 6: Send an event to the fraud detector
	SendFraudEvent(client)

	// Step 7: Run a test prediction
	GetFraudScore(client)

	fmt.Println("AWS Fraud Detector Setup Completed.")
}
