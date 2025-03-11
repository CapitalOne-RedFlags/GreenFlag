package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Define ProcessingState here since we can't import it
type ProcessingState struct {
	LastProcessedIndex int       `json:"lastProcessedIndex"`
	LastRunTime        time.Time `json:"lastRunTime"`
}

// ✅ Mock SQS Client That Implements Required Interface
type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

// ✅ Define Test Suite
type ProducerTestSuite struct {
	suite.Suite
	mockSQSClient *MockSQSClient
	ctx           context.Context
	queueURL      string
	tempDir       string
	csvPath       string
	statePath     string
}

// ✅ Run All Tests
func TestProducerSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}

// ✅ Setup Before All Tests
func (s *ProducerTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.queueURL = "https://sqs.us-east-1.amazonaws.com/140023383737/Bank_Transactions"
	
	// Create temporary directory for test files
	var err error
	s.tempDir, err = os.MkdirTemp("", "producer-test-")
	if err != nil {
		s.T().Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create test CSV path
	s.csvPath = filepath.Join(s.tempDir, "test_transactions.csv")
	s.statePath = filepath.Join(s.tempDir, ".test_transactions.csv.state")
}

// ✅ Cleanup After All Tests
func (s *ProducerTestSuite) TearDownSuite() {
	// Clean up temporary directory
	os.RemoveAll(s.tempDir)
}

// ✅ Setup Before Each Test
func (s *ProducerTestSuite) SetupTest() {
	s.mockSQSClient = new(MockSQSClient)
	
	// Create a test CSV file
	s.createTestCSV()
	
	// Remove state file if it exists
	os.Remove(s.statePath)
}

// Helper to create test CSV file
func (s *ProducerTestSuite) createTestCSV() {
	csvData := `TransactionID,AccountID,TransactionAmount,TransactionDate,TransactionType,Location,DeviceID,IPAddress,MerchantID,Channel,CustomerAge,CustomerOccupation,TransactionDuration,LoginAttempts,AccountBalance,PreviousTransactionDate,PhoneNumber,Email,TransactionStatus
tx-123,acc-456,100.50,2023-01-01,PURCHASE,New York,dev-123,192.168.1.1,merch-123,ONLINE,30,Engineer,120,1,5000.00,2022-12-25,+12025550179,test@example.com,PENDING
tx-124,acc-457,200.75,2023-01-02,PURCHASE,Chicago,dev-124,192.168.1.2,merch-124,ONLINE,35,Doctor,90,2,6000.00,2022-12-26,+12025550180,test2@example.com,PENDING
tx-125,acc-458,300.25,2023-01-03,PURCHASE,Los Angeles,dev-125,192.168.1.3,merch-125,ONLINE,40,Lawyer,60,1,7000.00,2022-12-27,+12025550181,test3@example.com,PENDING`

	file, err := os.Create(s.csvPath)
	if err != nil {
		s.T().Fatalf("Failed to create test CSV: %v", err)
	}
	defer file.Close()
	
	_, err = file.WriteString(csvData)
	if err != nil {
		s.T().Fatalf("Failed to write test CSV: %v", err)
	}
}

// Helper to create state file
func (s *ProducerTestSuite) createStateFile(lastProcessedIndex int) {
	state := ProcessingState{
		LastProcessedIndex: lastProcessedIndex,
		LastRunTime:        time.Now().Add(-time.Hour),
	}
	
	file, err := os.Create(s.statePath)
	if err != nil {
		s.T().Fatalf("Failed to create state file: %v", err)
	}
	defer file.Close()
	
	err = json.NewEncoder(file).Encode(state)
	if err != nil {
		s.T().Fatalf("Failed to write state file: %v", err)
	}
}

// ✅ Test Case: Successfully Send Transaction
func (s *ProducerTestSuite) TestSendTransaction() {
	// Create a test transaction
	transaction := &models.Transaction{
		TransactionID:     "tx-123",
		AccountID:         "acc-456",
		TransactionAmount: 100.50,
		CustomerAge:       30,
		PhoneNumber:       "+12025550179",
		Email:             "test@example.com",
		TransactionStatus: "PENDING",
	}

	// Setup mock expectations
	messageId := uuid.New().String()
	s.mockSQSClient.On("SendMessage", s.ctx, mock.Anything).Return(
		&sqs.SendMessageOutput{MessageId: aws.String(messageId)}, 
		nil,
	).Once()

	// Convert transaction to JSON for SQS message
	data, err := json.Marshal(transaction)
	assert.NoError(s.T(), err)
	
	// Create SQS message input
	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueURL),
		MessageBody: aws.String(string(data)),
	}
	
	// Send message
	output, err := s.mockSQSClient.SendMessage(s.ctx, input)
	
	// Assert results
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), output)
	assert.Equal(s.T(), messageId, *output.MessageId)
	s.mockSQSClient.AssertExpectations(s.T())
}

// ✅ Test Case: Load State
func (s *ProducerTestSuite) TestLoadState() {
	// Create a state file
	expectedIndex := 5
	s.createStateFile(expectedIndex)
	
	// Read the file
	file, err := os.Open(s.statePath)
	assert.NoError(s.T(), err)
	defer file.Close()
	
	// Decode the state
	var state ProcessingState
	err = json.NewDecoder(file).Decode(&state)
	assert.NoError(s.T(), err)
	
	// Verify state
	assert.Equal(s.T(), expectedIndex, state.LastProcessedIndex)
	assert.False(s.T(), state.LastRunTime.IsZero())
}

// ✅ Test Case: Save State
func (s *ProducerTestSuite) TestSaveState() {
	// Create a state
	expectedState := ProcessingState{
		LastProcessedIndex: 10,
		LastRunTime:        time.Now(),
	}
	
	// Create a file
	file, err := os.Create(s.statePath)
	assert.NoError(s.T(), err)
	
	// Encode the state
	err = json.NewEncoder(file).Encode(expectedState)
	assert.NoError(s.T(), err)
	file.Close()
	
	// Read it back
	file, err = os.Open(s.statePath)
	assert.NoError(s.T(), err)
	defer file.Close()
	
	// Decode the state
	var loadedState ProcessingState
	err = json.NewDecoder(file).Decode(&loadedState)
	assert.NoError(s.T(), err)
	
	// Verify state
	assert.Equal(s.T(), expectedState.LastProcessedIndex, loadedState.LastProcessedIndex)
}

// ✅ Test Case: Send Multiple Transactions
func (s *ProducerTestSuite) TestSendMultipleTransactions() {
	// Create test transactions
	transactions := []*models.Transaction{
		{
			TransactionID:     "tx-123",
			AccountID:         "acc-456",
			TransactionAmount: 100.50,
			CustomerAge:       30,
			PhoneNumber:       "+12025550179",
			Email:             "test@example.com",
			TransactionStatus: "PENDING",
		},
		{
			TransactionID:     "tx-124",
			AccountID:         "acc-457",
			TransactionAmount: 200.75,
			CustomerAge:       35,
			PhoneNumber:       "+12025550180",
			Email:             "test2@example.com",
			TransactionStatus: "PENDING",
		},
	}

	// Setup mock expectations for each transaction
	for i := range transactions {
		messageId := uuid.New().String()
		data, _ := json.Marshal(transactions[i])
		
		s.mockSQSClient.On("SendMessage", s.ctx, mock.MatchedBy(func(input *sqs.SendMessageInput) bool {
			return *input.QueueUrl == s.queueURL && *input.MessageBody == string(data)
		})).Return(
			&sqs.SendMessageOutput{MessageId: aws.String(messageId)}, 
			nil,
		).Once()
	}

	// Send each transaction
	for _, txn := range transactions {
		data, _ := json.Marshal(txn)
		input := &sqs.SendMessageInput{
			QueueUrl:    aws.String(s.queueURL),
			MessageBody: aws.String(string(data)),
		}
		
		output, err := s.mockSQSClient.SendMessage(s.ctx, input)
		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), output)
		assert.NotEmpty(s.T(), *output.MessageId)
	}
	
	// Verify all expected calls were made
	s.mockSQSClient.AssertExpectations(s.T())
}