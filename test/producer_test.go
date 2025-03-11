package test

import (
	"context"
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Mock SQS client
type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	args := m.Called(ctx, params)
	return &sqs.SendMessageOutput{
		MessageId: aws.String(uuid.New().String()),
	}, args.Error(1)
}

func (m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *MockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(ctx, params)
	return &sqs.DeleteMessageOutput{}, args.Error(1)
}

// ProducerTestSuite defines the test suite
type ProducerTestSuite struct {
	suite.Suite
	mockSQSClient *MockSQSClient
	sqsHandler    *messaging.SQSHandler
	ctx           context.Context
	queueURL      string
}

func TestProducerSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}

// SetupSuite runs before all tests
func (s *ProducerTestSuite) SetupSuite() {
	config.InitializeConfig()
	s.ctx = context.Background()
	s.queueURL = "https://sqs.us-east-1.amazonaws.com/140023383737/Bank_Transactions" //TODO: Change to the correct queue URL
}

// SetupTest runs before each test
func (s *ProducerTestSuite) SetupTest() {
	s.mockSQSClient = new(MockSQSClient)
	s.sqsHandler = messaging.NewSQSHandlerWithClient(s.mockSQSClient, s.queueURL)
}

// TestSendTransaction tests sending a single transaction
func (s *ProducerTestSuite) TestSendTransaction() {
	// Create a test transaction
	transaction := &models.Transaction{
		TransactionID:      "tx-123",
		AccountID:          "acc-456",
		TransactionAmount:  100.50,
		CustomerAge:        30,
		PhoneNumber:        "+12025550179",
		Email:              "test@example.com",
		TransactionStatus:  "PENDING",
	}

	// Setup mock expectations
	s.mockSQSClient.On("SendMessage", s.ctx, mock.Anything).Return(&sqs.SendMessageOutput{
		MessageId: aws.String("msg-789"),
	}, nil)

	// Call the method being tested
	err := s.sqsHandler.SendTransaction(s.ctx, transaction)
	
	// Assert results
	assert.NoError(s.T(), err)
	s.mockSQSClient.AssertExpectations(s.T())
}

// TestSendMultipleTransactions tests sending multiple transactions
func (s *ProducerTestSuite) TestSendMultipleTransactions() {
	// Create test CSV data
	csvData := `TransactionID,AccountID,TransactionAmount,CustomerAge,PhoneNumber,Email,TransactionStatus
tx-123,acc-456,100.50,30,+12025550179,test@example.com,PENDING
tx-124,acc-457,200.75,35,+12025550180,test2@example.com,PENDING
tx-125,acc-458,300.25,40,+12025550181,test3@example.com,PENDING`

	// Create a CSV reader
	reader := csv.NewReader(strings.NewReader(csvData))
	
	// Skip header
	_, err := reader.Read()
	assert.NoError(s.T(), err)
	
	// Setup mock expectations - expect 3 calls
	s.mockSQSClient.On("SendMessage", s.ctx, mock.Anything).Return(&sqs.SendMessageOutput{
		MessageId: aws.String(uuid.New().String()),
	}, nil).Times(3)
	
	// Process each record
	for {
		record, err := reader.Read()
		if err != nil {
			break // End of file
		}
		
		// Create transaction from CSV record
		transaction := &models.Transaction{
			TransactionID:      record[0],
			AccountID:          record[1],
			TransactionAmount:  100.0, // Simplified for test
			CustomerAge:        30,    // Simplified for test
			PhoneNumber:        record[4],
			Email:              record[5],
			TransactionStatus:  record[6],
		}
		
		// Send transaction
		err = s.sqsHandler.SendTransaction(s.ctx, transaction)
		assert.NoError(s.T(), err)
	}
	
	// Verify all expected calls were made
	s.mockSQSClient.AssertExpectations(s.T())
}

// TestSendTransactionError tests error handling
func (s *ProducerTestSuite) TestSendTransactionError() {
	// Create a test transaction
	transaction := &models.Transaction{
		TransactionID:      "tx-123",
		AccountID:          "acc-456",
		TransactionAmount:  100.50,
		CustomerAge:        30,
		PhoneNumber:        "+12025550179",
		Email:              "test@example.com",
		TransactionStatus:  "PENDING",
	}

	// Setup mock to return an error
	s.mockSQSClient.On("SendMessage", s.ctx, mock.Anything).Return(nil, assert.AnError)

	// Call the method being tested
	err := s.sqsHandler.SendTransaction(s.ctx, transaction)
	
	// Assert error was returned
	assert.Error(s.T(), err)
	s.mockSQSClient.AssertExpectations(s.T())
}

// TestIntegrationWithRealSQS tests with a real SQS queue (optional)
func (s *ProducerTestSuite) TestIntegrationWithRealSQS() {
	// Skip this test unless explicitly enabled
	if os.Getenv("ENABLE_SQS_INTEGRATION_TESTS") != "true" {
		s.T().Skip("Skipping integration test. Set ENABLE_SQS_INTEGRATION_TESTS=true to run")
	}

	// Load AWS config
	awsConf, err := config.LoadAWSConfig(s.ctx)
	assert.NoError(s.T(), err)
	
	// Create real SQS client
	sqsClient := sqs.NewFromConfig(awsConf.Config)
	realHandler := messaging.NewSQSHandler(sqsClient, config.SQSConfig.QueueURL)
	
	// Create a test transaction
	transaction := &models.Transaction{
		TransactionID:      "integration-" + uuid.New().String(),
		AccountID:          "acc-" + uuid.New().String(),
		TransactionAmount:  100.50,
		CustomerAge:        30,
		PhoneNumber:        "+12025550179",
		Email:              "test@example.com",
		TransactionStatus:  "PENDING",
	}
	
	// Send transaction
	err = realHandler.SendTransaction(s.ctx, transaction)
	assert.NoError(s.T(), err)
	
	// Verify message was sent by receiving it
	output, err := sqsClient.ReceiveMessage(s.ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(config.SQSConfig.QueueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     5,
	})
	assert.NoError(s.T(), err)
	
	// Clean up - delete any received messages
	for _, msg := range output.Messages {
		_, err = sqsClient.DeleteMessage(s.ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(config.SQSConfig.QueueURL),
			ReceiptHandle: msg.ReceiptHandle,
		})
		assert.NoError(s.T(), err)
	}
}
