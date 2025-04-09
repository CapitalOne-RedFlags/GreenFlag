package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/joho/godotenv"
)

const projectDirName = "GreenFlag" // Your project name

type TwilioSecrets struct {
	Username string `json:"TWILIO_USERNAME"`
	Password string `json:"TWILIO_PASSWORD"`
}

// LoadEnv loads environment variables from a .env file
func LoadEnv() {
	// Find project root directory dynamically
	projectName := regexp.MustCompile(`^(.*` + projectDirName + `)`)
	currentWorkDirectory, _ := os.Getwd()
	rootPath := projectName.Find([]byte(currentWorkDirectory))

	// Try to load .env from project root
	err := godotenv.Load(string(rootPath) + `/.env`)
	if err != nil {
		log.Printf("Warning: Could not load .env file from project root: %v", err)

		// Fallback to current directory
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: No .env file found in current directory")
		}
	} else {
		log.Printf("Loaded environment from %s/.env", string(rootPath))
	}

	// Print environment variables for debugging
	log.Printf("AWS Region: %s", GetEnv("AWS_REGION", "us-east-1"))
	log.Printf("SQS Queue URL: %s", GetEnv("SQS_QUEUE_URL", ""))
	log.Printf("DynamoDB Table: %s", GetEnv("DYNAMODB_TABLE", "Transactions"))
	log.Printf("DynamoDB Endpoint: %s", GetEnv("DYNAMODB_ENDPOINT", ""))
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
	TopicName      string
	TwilioUsername string
	TwilioPassword string
}{}

// SQSConfig stores SQS-specific configurations
var SQSConfig = &struct {
	QueueURL string
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

	// Initialize SQS config
	SQSConfig.QueueURL = GetEnv("QUEUE_URL", "")

	// Initialize SNS config
	SNSMessengerConfig.TopicName = GetEnv("SNS_TOPIC", "FraudAlerts")
	secrets, err := LoadTwilioSecrets("greenflags/twilio")
	if err != nil {
		log.Printf("error loading Twilio secrets:", err)
	} else {
		SNSMessengerConfig.TwilioUsername = secrets.Username
		SNSMessengerConfig.TwilioPassword = secrets.Password
	}

	log.Printf("DynamoDB Table: %s", DBConfig.TableName)
	log.Printf("DynamoDB Endpoint: %s", DBConfig.DynamoDBEndpoint)
	log.Printf("AWS Region: %s", GetEnv("AWS_REGION", "us-east-1"))
	log.Printf("SQS Queue URL: %s", SQSConfig.QueueURL)
	log.Printf("CI Mode: %s", GetEnv("CI", "false"))

}

func IsCI() bool {
	return GetEnv("CI", "false") == "true"
}

func PrintDBConfig() {
	fmt.Printf("DynamoDB Table: %s\n", DBConfig.TableName)
	fmt.Printf("DynamoDB Endpoint: %s\n", DBConfig.DynamoDBEndpoint)
	fmt.Printf("AWS Region: %s\n", GetEnv("AWS_REGION", "us-east-1"))
	fmt.Printf("SQS Queue URL: %s\n", SQSConfig.QueueURL)
	fmt.Printf("CI Mode: %v\n", IsCI())
}

func LoadTwilioSecrets(secretName string) (*TwilioSecrets, error) {
	region := "us-east-1"

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	svc := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // default stage
	}

	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret: %w", err)
	}

	var twilio TwilioSecrets
	if err := json.Unmarshal([]byte(*result.SecretString), &twilio); err != nil {
		return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	return &twilio, nil
}
