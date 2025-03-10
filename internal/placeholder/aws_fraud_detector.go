package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector/types"
)

func main() {
	// Load AWS SDK config
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Create Fraud Detector client
	client := frauddetector.NewFromConfig(cfg)

	// Create a new detector
	detectorID := "my-fraud-detector"
	_, err = client.CreateDetectorVersion(context.TODO(), &frauddetector.CreateDetectorVersionInput{
		DetectorId:      aws.String(detectorID),
		RuleExecutionMode: types.RuleExecutionModeFirstMatched,
		Description:     aws.String("Fraud detection model"),
		Rules: []types.Rule{
			{
				DetectorId: aws.String(detectorID),
				RuleId:     aws.String("rule-1"),
				RuleVersion: aws.String("1"),
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create detector version: %v", err)
	}

	fmt.Println("AWS Fraud Detector setup complete.")
}
