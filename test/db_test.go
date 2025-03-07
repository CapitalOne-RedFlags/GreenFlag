package test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/db"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TransactionRepositoryTestSuite defines the test suite
type TransactionRepositoryTestSuite struct {
	suite.Suite
	dbClient   *db.DynamoDBClient
	repository *db.TransactionRepository
	tableName  string
	ctx        context.Context
}

func TestTransactionRepositorySuite(t *testing.T) {
	suite.Run(t, new(TransactionRepositoryTestSuite))
}

// SetupSuite runs before all tests
func (s *TransactionRepositoryTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Initialize configurations

	// Skip test in CI if no endpoint is configured
	if config.IsCI() && config.DBConfig.DynamoDBEndpoint == "http://localhost:8000" {
		s.T().Skip("Skipping DynamoDB integration tests in CI: No endpoint configured.")
		return
	}

	// Print config for debugging
	config.PrinDBConfig()

	// Generate a unique table name per test run
	s.tableName = fmt.Sprintf("%s-%s", config.DBConfig.TableName, uuid.New().String())

	awsConf, err := config.LoadAWSConfig(context.Background())
	if err != nil {
		log.Fatalf("Failed to initialize AWS config: %v", err)
	}
	s.dbClient = db.NewDynamoDBClient(dynamodb.NewFromConfig(awsConf.Config), s.tableName)
	s.repository = db.NewTransactionRepository(s.dbClient)

	// Update config for this test instance
	config.DBConfig.TableName = s.tableName

	// Create test table
	s.createTestTable(s.dbClient.Client)
}

// Create a test table for DynamoDB
func (s *TransactionRepositoryTestSuite) createTestTable(client *dynamodb.Client) {
	_, err := client.CreateTable(s.ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(s.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(config.DBConfig.Keys.PartitionKey),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String(config.DBConfig.Keys.SortKey),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(config.DBConfig.Keys.PartitionKey),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String(config.DBConfig.Keys.SortKey),
				KeyType:       types.KeyTypeRange,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	// Ignore existing table errors
	if err != nil && !strings.Contains(err.Error(), "ResourceInUseException") {
		s.T().Fatalf("Failed to create test table: %v", err)
	}

	// Wait until table is active
	waiter := dynamodb.NewTableExistsWaiter(client)
	err = waiter.Wait(s.ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	}, 30*time.Second)
	if err != nil {
		s.T().Fatalf("Failed waiting for table to be active: %v", err)
	}
}

// TearDownSuite runs after all tests
func (s *TransactionRepositoryTestSuite) TearDownSuite() {
	if s.dbClient == nil {
		return
	}
	_, err := s.dbClient.Client.DeleteTable(s.ctx, &dynamodb.DeleteTableInput{
		TableName: aws.String(s.tableName),
	})
	if err != nil {
		s.T().Logf("Failed to delete test table %s: %v", s.tableName, err)
	}
}


// createValidTransaction creates a valid sample transaction for tests
func (s *TransactionRepositoryTestSuite) createValidTransaction() models.Transaction {
	return models.Transaction{
		TransactionID:           uuid.New().String(),
		AccountID:               "TEST-" + uuid.New().String(),
		TransactionAmount:       100.50,
		TransactionDate:         time.Now().Format(time.RFC3339),
		TransactionType:         "PURCHASE",
		Location:                "New York",
		DeviceID:                "device-123",
		IPAddress:               "192.168.1.1",
		MerchantID:              "merchant-456",
		Channel:                 "WEB",
		CustomerAge:             30,
		CustomerOccupation:      "Engineer",
		TransactionDuration:     120,
		LoginAttempts:           1,
		AccountBalance:          5000.00,
		PreviousTransactionDate: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		PhoneNumber:             "+12025550179",
		Email:                   "test@example.com",
		TransactionStatus:       "PENDING",
	}
}

func (s *TransactionRepositoryTestSuite) TestSaveTransaction() {
	// s.T().Parallel() // Uncomment if you want to allow parallel test runs

	transaction := s.createValidTransaction()

	// Save the transaction
	output, metadata, err := s.repository.SaveTransaction(s.ctx, &transaction)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), output)
	assert.NotEmpty(s.T(), metadata)

	// Try saving the same transaction again; expect an error
	_, _, err = s.repository.SaveTransaction(s.ctx, &transaction)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "transaction already exists")
}

func (s *TransactionRepositoryTestSuite) TestGetTransaction() {
	// s.T().Parallel()

	transaction := s.createValidTransaction()
	_, _, err := s.repository.SaveTransaction(s.ctx, &transaction)
	assert.NoError(s.T(), err)

	// Get the transaction
	retrieved, err := s.repository.GetTransaction(s.ctx, transaction.AccountID, transaction.TransactionID)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), retrieved)
	assert.Equal(s.T(), transaction.TransactionID, retrieved.TransactionID)
	assert.Equal(s.T(), transaction.AccountID, retrieved.AccountID)
	assert.Equal(s.T(), transaction.TransactionAmount, retrieved.TransactionAmount)

	// Attempt to get a non-existent item
	_, err = s.repository.GetTransaction(s.ctx, "non-existent", "non-existent")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "item not found")
}

func (s *TransactionRepositoryTestSuite) TestUpdateTransaction() {
	// s.T().Parallel()

	transaction := s.createValidTransaction()
	_, _, err := s.repository.SaveTransaction(s.ctx, &transaction)
	assert.NoError(s.T(), err)

	updateData := models.Transaction{
		TransactionAmount: 200.75,
		Location:          "Los Angeles",
		TransactionStatus: "APPROVED",
	}

	result, err := s.repository.UpdateTransaction(s.ctx, transaction.AccountID, transaction.TransactionID, &updateData)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)

	// Verify updated fields
	updated, err := s.repository.GetTransaction(s.ctx, transaction.AccountID, transaction.TransactionID)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 200.75, updated.TransactionAmount)
	assert.Equal(s.T(), "Los Angeles", updated.Location)
	assert.Equal(s.T(), "APPROVED", updated.TransactionStatus)
}

func (s *TransactionRepositoryTestSuite) TestDeleteTransaction() {
	// s.T().Parallel()

	transaction := s.createValidTransaction()
	_, _, err := s.repository.SaveTransaction(s.ctx, &transaction)
	assert.NoError(s.T(), err)

	err = s.repository.DeleteTransaction(s.ctx, transaction.AccountID, transaction.TransactionID)
	assert.NoError(s.T(), err)

	// Verify it's gone
	_, err = s.repository.GetTransaction(s.ctx, transaction.AccountID, transaction.TransactionID)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "item not found")
}

func (s *TransactionRepositoryTestSuite) TestInvalidInput() {
	// s.T().Parallel()

	// Missing required fields
	invalid := models.Transaction{TransactionAmount: 100.50}
	_, _, err := s.repository.SaveTransaction(s.ctx, &invalid)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation failed")

	// Empty parameters in Get
	_, err = s.repository.GetTransaction(s.ctx, "", "")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot be empty")

	// Empty parameters in Update
	_, err = s.repository.UpdateTransaction(s.ctx, "", "", &models.Transaction{})
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot be empty")

	// Empty parameters in Delete
	err = s.repository.DeleteTransaction(s.ctx, "", "")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot be empty")
}
