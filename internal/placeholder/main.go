package main

import "fmt"

func main() {
	fmt.Println("Starting AWS Fraud Detector Setup...")

	client := GetAWSConfig()

	// Set up entity types and event types
	CreateEntityType(client)
	CreateEventType(client)

	// Create the fraud detector
	CreateDetector(client)

	// Send an event to the fraud detector
	SendFraudEvent(client)

	fmt.Println("AWS Fraud Detector Setup Completed.")
}
