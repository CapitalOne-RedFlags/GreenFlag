package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from a .env file
func LoadEnv() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("Warning: No .env file found or failed to load.")
	}

	// log.Printf("DEBUG: AWS_ACCESS_KEY_ID=%s\n", os.Getenv("AWS_ACCESS_KEY_ID"))
	// log.Printf("DEBUG: AWS_SECRET_ACCESS_KEY=%s\n", os.Getenv("AWS_SECRET_ACCESS_KEY"))
	// log.Printf("DEBUG: AWS_SESSION_TOKEN=%s\n", os.Getenv("AWS_SESSION_TOKEN")) // Optional
	// log.Printf("DEBUG: AWS_REGION=%s\n", os.Getenv("AWS_REGION"))
}

// DBConfig stores table settings
var DBConfig = &struct {
	TableName           string
	DynamoDBEndpoint    string
	AllowedUpdateFields map[string]bool
	UpdateCondition     string
	Keys                struct {
		PartitionKey string
		SortKey      string
	}
}{}

var SNSMessengerConfig = &struct {
	TopicName string
}{}

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

// InitializeConfig initializes the configuration by loading environment variables
func InitializeConfig() {
	LoadEnv() // Load .env variables

	DBConfig.TableName = GetEnv("DYNAMODB_TABLE_NAME", "TestTransactionsTable")
	DBConfig.DynamoDBEndpoint = GetEnv("DYNAMODB_ENDPOINT", "http://localhost:8000")
	DBConfig.AllowedUpdateFields = map[string]bool{
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
	}
	DBConfig.UpdateCondition = "TransactionStatus = Pending"
	DBConfig.Keys = struct {
		PartitionKey string
		SortKey      string
	}{
		PartitionKey: "AccountID",
		SortKey:      "TransactionID",
	}

	SNSMessengerConfig.TopicName = GetEnv("SNS_TOPIC", "FraudAlerts")
}

func IsCI() bool {
	return GetEnv("CI", "false") == "true"
}

func PrintDBConfig() {
	fmt.Printf("DynamoDB Table: %s\n", DBConfig.TableName)
	fmt.Printf("DynamoDB Endpoint: %s\n", DBConfig.DynamoDBEndpoint)
	fmt.Printf("AWS Region: %s\n", GetEnv("AWS_REGION", "us-east-1"))
	fmt.Printf("CI Mode: %v\n", IsCI())
}
