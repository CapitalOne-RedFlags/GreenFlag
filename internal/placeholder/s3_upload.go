package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// UploadToS3 uploads a file to an S3 bucket.
func UploadToS3(bucket, key, filePath string) {
	cfg := GetAWSConfig()

	client := s3.NewFromConfig(cfg)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		log.Fatalf("Failed to upload file to S3: %v", err)
	}
	fmt.Println("File uploaded to S3 successfully.")
}
