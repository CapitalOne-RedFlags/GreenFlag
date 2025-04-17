package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/frauddetector"
)

// TrainModel starts model training.
func TrainModel(client *frauddetector.Client) {
	modelID := "fraud_model"

	_, err := client.CreateModelVersion(context.TODO(), &frauddetector.CreateModelVersionInput{
		ModelId:            aws.String(modelID),
		ModelType:          frauddetector.ModelTypeOnlineFraudInsights,
		TrainingDataSource: aws.String("S3"),
		TrainingDataSchema: &frauddetector.TrainingDataSchema{
			TargetAttributeName: aws.String("IS_FRAUD"),
		},
	})
	if err != nil {
		log.Fatalf("Failed to train model: %v", err)
	}
	fmt.Println("Model training started.")
}
