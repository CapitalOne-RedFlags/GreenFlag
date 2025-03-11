package test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
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

// ✅ Define Test Suite
type TransactionServiceTestSuite struct {
	suite.Suite
	testRepo TransactionRepositoryTestSuite
	mockRepo *MockTransactionRepository
	ctx      context.Context
}

// 🔄 Setup Before Each Test
func (suite *TransactionServiceTestSuite) SetupTest() {
	suite.mockRepo = &MockTransactionRepository{}
	suite.ctx = context.Background()
}

// ✅ Test Case: Successfully Saves Valid Transactions
func (suite *TransactionServiceTestSuite) TestTransactionService_Success() {

	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", CustomerAge: 26, TransactionAmount: 100.50, PhoneNumber: "+12025550179", Email: "test@example.com"},
		{TransactionID: "tx2", AccountID: "acc456", CustomerAge: 26, TransactionAmount: 200.75, PhoneNumber: "+12025550178", Email: "user@example.com"},
	}

	// ✅ Mock successful saves
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "tx1", nil).Once()
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[1]).Return(nil, "tx2", nil).Once()

	// ✅ Call `TransactionService`
	err := services.TransactionService(suite.ctx, transactions, suite.mockRepo)

	// ✅ Verify expected calls
	suite.mockRepo.AssertExpectations(suite.T())
	assert.NoError(suite.T(), err, "Should not return an error when transaction is sucessfully saved")
}

// Test Case: Save Fails Due to DynamoDB Error
func (suite *TransactionServiceTestSuite) TestTransactionService_SaveError() {
	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", TransactionAmount: 100.50, CustomerAge: 26, PhoneNumber: "+12025550179", Email: "test@example.com"},
	}

	// ✅ Mock a failure
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "", errors.New("DynamoDB error")).Once()

	// ✅ Call `TransactionService`
	err := services.TransactionService(suite.ctx, transactions, suite.mockRepo)

	// ✅ Ensure expected calls were made
	suite.mockRepo.AssertExpectations(suite.T())
	assert.Error(suite.T(), err, "Should return an error when transaction is sucessfully saved")
}

func (suite *TransactionServiceTestSuite) TestTransactionService_NoTransaction() {
	var transactions []models.Transaction
	// Call `TransactionService`
	err := services.TransactionService(suite.ctx, transactions, suite.mockRepo)

	// Ensure expected calls were made
	suite.mockRepo.AssertExpectations(suite.T())
	assert.NoError(suite.T(), err, "Should not return an error when no transactions are saved")
}

func (suite *TransactionServiceTestSuite) TestTransactionService_ParitalFail() {

	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", CustomerAge: 26, TransactionAmount: 100.50, PhoneNumber: "+12025550179", Email: "test@example.com"},
		{TransactionID: "tx2", AccountID: "acc456", CustomerAge: 26, TransactionAmount: 200.75, PhoneNumber: "+12025550178", Email: "user@example.com"},
	}

	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "tx1", errors.New("DynamoDB error")).Once()
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[1]).Return(nil, "tx2", nil).Once()

	err := services.TransactionService(suite.ctx, transactions, suite.mockRepo)

	suite.mockRepo.AssertExpectations(suite.T())
	assert.Error(suite.T(), err, "Should  return an error when transaction is partially saved")
	assert.Len(suite.T(), strings.Split(err.Error(), "\n"), 1)

}

func (suite *TransactionServiceTestSuite) TestTransactionService_MultipuleFailures() {

	transactions := []models.Transaction{
		{TransactionID: "tx1", AccountID: "acc123", CustomerAge: 26, TransactionAmount: 100.50, PhoneNumber: "+12025550179", Email: "test@example.com"},
		{TransactionID: "tx2", AccountID: "acc456", CustomerAge: 26, TransactionAmount: 200.75, PhoneNumber: "+12025550178", Email: "user@example.com"},
	}

	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[0]).Return(nil, "tx1", errors.New("DynamoDB error")).Once()
	suite.mockRepo.On("SaveTransaction", suite.ctx, &transactions[1]).Return(nil, "tx2", errors.New("DynamoDB error")).Once()

	err := services.TransactionService(suite.ctx, transactions, suite.mockRepo)

	suite.mockRepo.AssertExpectations(suite.T())
	assert.Error(suite.T(), err, "Should  return an error when transaction is partially saved")
	assert.Len(suite.T(), strings.Split(err.Error(), "\n"), 2)

}

func (suite *TransactionServiceTestSuite) TestTransactionService_integration() {
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

	serviceErr := services.TransactionService(suite.ctx, testTransaction, suite.testRepo.repository)
	suite.mockRepo.AssertExpectations(suite.T())
	assert.NoError(suite.T(), serviceErr, "Should not return an error when transactions are saved")

	res, getTransErr := suite.testRepo.repository.GetTransaction(suite.ctx, testTransaction[0].AccountID, testTransaction[0].TransactionID)
	assert.NoError(suite.T(), getTransErr, "Should not return an error when transactions are saved")
	assert.NotNil(suite.T(), res)
	assert.Equal(suite.T(), testTransaction[0], *res)
}

// Run All Tests
func TestTransactionServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionServiceTestSuite))
}
