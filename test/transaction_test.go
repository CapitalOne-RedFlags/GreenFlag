package test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// ✅ Mock Repository That Implements `db.TransactionRepository`
type MockTransactionRepository struct {
	mock.Mock
}

// GetFraudTransaction implements db.TransactionRepository.
func (m *MockTransactionRepository) GetTransactionByNumberAndStatus(ctx context.Context, phoneNumber string, status string) ([]models.Transaction, error) {
	args := m.Called(ctx, phoneNumber)
	return nil, args.Error(1)
}

// UpdateFraudTransaction implements db.TransactionRepository.
func (m *MockTransactionRepository) UpdateFraudTransaction(ctx context.Context, phoneNumber string, isFraud bool, status string) (int, error) {
	args := m.Called(ctx, phoneNumber, isFraud)
	return 1, args.Error(1)
}

// ✅ Implement `SaveTransaction`
func (m *MockTransactionRepository) SaveTransaction(ctx context.Context, txn *models.Transaction) (*dynamodb.PutItemOutput, string, error) {
	args := m.Called(ctx, txn)
	return nil, args.String(1), args.Error(2)
}

// ✅ Implement `GetTransaction`
func (m *MockTransactionRepository) GetTransaction(ctx context.Context, accountID, transactionID string) (*models.Transaction, error) {
	args := m.Called(ctx, accountID, transactionID)
	return nil, args.Error(1)
}

// ✅ Implement `UpdateTransaction`
func (m *MockTransactionRepository) UpdateTransaction(ctx context.Context, accountID, transactionID string, values *models.Transaction) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, accountID, transactionID, values)
	return nil, args.Error(1)
}

// ✅ Implement `DeleteTransaction`
func (m *MockTransactionRepository) DeleteTransaction(ctx context.Context, accountID, transactionID string) error {
	args := m.Called(ctx, accountID, transactionID)
	return args.Error(0)
}

type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) TransactionService(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, error) {
	args := m.Called(ctx, transactions)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

// ✅ Define Test Suite
type TransactionPipelineTestSuite struct {
	suite.Suite
	testRepo               TransactionRepositoryTestSuite
	mockRepo               *MockTransactionRepository
	mockTransactionService *MockTransactionService
	ctx                    context.Context
}

// 🔄 Setup Before Each Test
func (suite *TransactionPipelineTestSuite) SetupTest() {
	suite.mockRepo = &MockTransactionRepository{}
	suite.mockTransactionService = &MockTransactionService{}
	suite.ctx = context.Background()
}

// ✅ Test Case: Successfully Saves Valid Transactions
func (suite *TransactionPipelineTestSuite) TestTransactionService_Success() {

	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", CustomerAge: 26, TransactionAmount: 100.50, PhoneNumber: "+12025550179", Email: "test@example.com"},
		{TransactionID: "tx2", AccountID: "acc456", CustomerAge: 26, TransactionAmount: 200.75, PhoneNumber: "+12025550178", Email: "user@example.com"},
	}

	// ✅ Mock successful saves
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "tx1", nil).Once()
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[1]).Return(nil, "tx2", nil).Once()

	// ✅ Call `TransactionService`
	service := services.NewTransactionService(suite.mockRepo)
	failedTransactions, err := service.TransactionService(suite.ctx, transactions)

	// ✅ Verify expected calls
	suite.mockRepo.AssertExpectations(suite.T())
	assert.NoError(suite.T(), err, "Should not return an error when transaction is sucessfully saved")
	assert.Empty(suite.T(), failedTransactions)
}

// Test Case: Save Fails Due to DynamoDB Error
func (suite *TransactionPipelineTestSuite) TestTransactionService_SaveError() {
	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", TransactionAmount: 100.50, CustomerAge: 26, PhoneNumber: "+12025550179", Email: "test@example.com"},
	}

	// ✅ Mock a failure
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "", errors.New("DynamoDB error")).Once()

	// ✅ Call `TransactionService`
	service := services.NewTransactionService(suite.mockRepo)
	failedTransactions, err := service.TransactionService(suite.ctx, transactions)

	// ✅ Ensure expected calls were made
	suite.mockRepo.AssertExpectations(suite.T())
	assert.Error(suite.T(), err, "Should return an error when transaction is sucessfully saved")
	assert.Len(suite.T(), failedTransactions, 1)
}

func (suite *TransactionPipelineTestSuite) TestTransactionService_NoTransaction() {
	var transactions []models.Transaction
	// Call `TransactionService`
	service := services.NewTransactionService(suite.mockRepo)
	failedTransactions, err := service.TransactionService(suite.ctx, transactions)

	// Ensure expected calls were made
	suite.mockRepo.AssertExpectations(suite.T())
	assert.NoError(suite.T(), err, "Should not return an error when no transactions are saved")
	assert.Empty(suite.T(), failedTransactions)
}

func (suite *TransactionPipelineTestSuite) TestTransactionService_ParitalFail() {

	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", CustomerAge: 26, TransactionAmount: 100.50, PhoneNumber: "+12025550179", Email: "test@example.com"},
		{TransactionID: "tx2", AccountID: "acc456", CustomerAge: 26, TransactionAmount: 200.75, PhoneNumber: "+12025550178", Email: "user@example.com"},
	}

	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "tx1", errors.New("DynamoDB error")).Once()
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[1]).Return(nil, "tx2", nil).Once()

	service := services.NewTransactionService(suite.mockRepo)
	failedTransactions, err := service.TransactionService(suite.ctx, transactions)

	suite.mockRepo.AssertExpectations(suite.T())
	assert.Error(suite.T(), err, "Should  return an error when transaction is partially saved")
	assert.Len(suite.T(), strings.Split(err.Error(), "\n"), 1)
	assert.Len(suite.T(), failedTransactions, 1)
}

func (suite *TransactionPipelineTestSuite) TestTransactionService_MultipuleFailures() {

	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", CustomerAge: 26, TransactionAmount: 100.50, PhoneNumber: "+12025550179", Email: "test@example.com"},
		{TransactionID: "tx2", AccountID: "acc456", CustomerAge: 26, TransactionAmount: 200.75, PhoneNumber: "+12025550178", Email: "user@example.com"},
	}

	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "tx1", errors.New("DynamoDB error")).Once()
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[1]).Return(nil, "tx2", errors.New("DynamoDB error")).Once()

	service := services.NewTransactionService(suite.mockRepo)
	failedTransactions, err := service.TransactionService(suite.ctx, transactions)

	suite.mockRepo.AssertExpectations(suite.T())
	assert.Error(suite.T(), err, "Should  return an error when transaction is partially saved")
	assert.Len(suite.T(), strings.Split(err.Error(), "\n"), 2)
	assert.Len(suite.T(), failedTransactions, 2)
}

func (suite *TransactionPipelineTestSuite) TestTransactionService_integration() {
	suite.testRepo.SetupSuite()

	var testTransaction []models.Transaction
	testTransaction = append(testTransaction,
		models.Transaction{
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
		},
	)

	service := services.NewTransactionService(suite.testRepo.repository)
	failedTransactions, serviceErr := service.TransactionService(suite.ctx, testTransaction)

	suite.mockRepo.AssertExpectations(suite.T())
	assert.NoError(suite.T(), serviceErr, "Should not return an error when transactions are saved")
	assert.Empty(suite.T(), failedTransactions)

	res, getTransErr := suite.testRepo.repository.GetTransaction(suite.ctx, testTransaction[0].AccountID, testTransaction[0].TransactionID)
	assert.NoError(suite.T(), getTransErr, "Should not return an error when transactions are saved")
	assert.NotNil(suite.T(), res)
	assert.Equal(suite.T(), testTransaction[0], *res)
}

func (suite *TransactionPipelineTestSuite) TestTransactionHandler_PartialBatchFailure() {
	// Arrange
	testTxn1 := GetTestTransaction("test@example.com")
	testTxn2 := GetTestTransaction("jpoconnell4@wisc.edu")
	testTxn3 := GetTestTransaction("test@example.com")
	serviceArgs := []models.Transaction{testTxn1, testTxn2, testTxn3}

	eventRecord1 := getSQSEventRecord(testTxn1)
	eventRecord2 := getSQSEventRecord(testTxn2)
	eventRecord3 := getSQSEventRecord(testTxn3)

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			eventRecord1,
			eventRecord2,
			eventRecord3,
		},
	}

	suite.mockTransactionService.On(
		"TransactionService",
		mock.Anything,
		serviceArgs,
	).Return(
		[]models.Transaction{testTxn1, testTxn2},
		errors.New("test"),
	).Once()

	expectedRIDs := []string{eventRecord1.MessageId, eventRecord2.MessageId}
	handler := handlers.NewTransactionProcessingHandler(suite.mockTransactionService)

	// Act
	batchResult, err := handler.TransactionProcessingHandler(context.TODO(), event)

	// Assert
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), batchResult)
	assert.Len(suite.T(), batchResult.BatchItemFailures, 2)
	assert.ElementsMatch(suite.T(), batchResult.GetRids(), expectedRIDs)
	suite.mockTransactionService.AssertExpectations(suite.T())
}

func (suite *TransactionPipelineTestSuite) TestTransactionHandler_ShouldPartialFail_WithInvalidTransactionBody() {
	// Arrange
	testTxn1 := GetTestTransaction("test@example.com")
	testTxn2 := GetTestTransaction("jpoconnell4@wisc.edu")
	testTxn3 := GetTestTransaction("test@wisc.edu")
	serviceArgs := []models.Transaction{testTxn1, testTxn2}

	eventRecord1 := getSQSEventRecord(testTxn1)
	eventRecord2 := getSQSEventRecord(testTxn2)
	eventRecord3 := getSQSEventRecord(testTxn3)

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			eventRecord1,
			eventRecord2,
			events.SQSMessage{
				MessageId: eventRecord3.MessageId,
				Body:      "Bad Transaction Body",
			},
		},
	}

	suite.mockTransactionService.On(
		"TransactionService",
		mock.Anything,
		serviceArgs,
	).Return(
		[]models.Transaction{},
		nil,
	).Once()

	expectedRIDs := []string{eventRecord3.MessageId}
	handler := handlers.NewTransactionProcessingHandler(suite.mockTransactionService)

	// Act
	batchResult, err := handler.TransactionProcessingHandler(context.TODO(), event)

	// Assert
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), batchResult)
	assert.Len(suite.T(), batchResult.BatchItemFailures, 1)
	assert.ElementsMatch(suite.T(), batchResult.GetRids(), expectedRIDs)
	suite.mockTransactionService.AssertExpectations(suite.T())
}

// Run All Tests
func TestTransactionPipelineTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionPipelineTestSuite))
}
