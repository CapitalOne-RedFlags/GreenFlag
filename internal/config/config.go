package config

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// DBConfig stores table settings
var DBConfig = &struct {
	TableName           string
	DynamoDBEndpoint    string
	AllowedUpdateFields map[string]bool
	UpdateCondition     string
	Keys               struct {
		PartitionKey string
		SortKey      string
	}
}{
	TableName:        GetEnv("DYNAMODB_TABLE_NAME", "TransactionsTable"),
	DynamoDBEndpoint: GetEnv("DYNAMODB_ENDPOINT", "http://localhost:8000"),
	AllowedUpdateFields: map[string]bool{
		"TransactionStatus":       true,
		"TransactionAmount":       true,
		"TransactionDate":         true,
		"Location":                true,
		"DeviceID":                true,
		"IPAddress":               true,
		"MerchantID":              true,
		"Channel":                 true,
		"CustomerAge":             true,
		"CustomerOccupation":      true,
		"TransactionDuration":     true,
		"LoginAttempts":           true,
		"AccountBalance":          true,
		"PreviousTransactionDate": true,
		"PhoneNumber":             true,
		"Email":                   true,
	},
	UpdateCondition: "TransactionStatus = :pending",
	Keys: struct {
		PartitionKey string
		SortKey      string
	}{
		PartitionKey: "AccountID",
		SortKey:      "TransactionID",
	},
}

// AWSConfig stores AWS-specific configurations
type AWSConfig struct {
	Region      string
	Credentials aws.Credentials
	Config      aws.Config
}

// GetEnv retrieves an environment variable or returns a default value
func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// LoadAWSConfig initializes and returns a new AWSConfig instance
func LoadAWSConfig(ctx context.Context) (*AWSConfig, error) {
	region := GetEnv("AWS_REGION", "us-east-1")

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     GetEnv("AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: GetEnv("AWS_SECRET_ACCESS_KEY", ""),
				SessionToken:    GetEnv("AWS_SESSION_TOKEN", ""),
			},
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &AWSConfig{
		Region:      region,
		Credentials: creds,
		Config:      cfg,
	}, nil
}

func IsCI() bool {
	return GetEnv("CI", "false") == "true"
}

func PrinDBConfig() {
	fmt.Printf("DynamoDB Table: %s\n", DBConfig.TableName)
	fmt.Printf("DynamoDB Endpoint: %s\n", DBConfig.DynamoDBEndpoint)
	fmt.Printf("AWS Region: %s\n", GetEnv("AWS_REGION", "us-east-1"))
	fmt.Printf("CI Mode: %v\n", IsCI())
}
