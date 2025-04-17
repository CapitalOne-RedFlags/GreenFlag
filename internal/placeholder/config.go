package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
)

// GetAWSConfig initializes the AWS configuration and returns a Fraud Detector client.
func GetAWSConfig() *frauddetector.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}
	return frauddetector.NewFromConfig(cfg)
}
