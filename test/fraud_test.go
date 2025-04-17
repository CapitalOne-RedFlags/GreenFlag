package test

import (
	"context"
	"errors"
	"testing"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockEventDispatcher struct {
	mock.Mock
}

// DeleteTransaction implements db.TransactionRepository.
func (m *MockEventDispatcher) DeleteTransaction(ctx context.Context, accountID string, transactionID string) error {
	args := m.Called(ctx, accountID, transactionID)
	return args.Error(0)
}

// GetFraudTransaction implements db.TransactionRepository.
func (m *MockEventDispatcher) GetTransactionByNumberAndStatus(ctx context.Context, phoneNumber string, status string) ([]models.Transaction, error) {
	args := m.Called(ctx, phoneNumber, status)
	return nil, args.Error(1)
}

// GetTransaction implements db.TransactionRepository.
func (m *MockEventDispatcher) GetTransaction(ctx context.Context, accountID string, transactionID string) (*models.Transaction, error) {
	args := m.Called(ctx, accountID, transactionID)
	return nil, args.Error(1)
}

// SaveTransaction implements db.TransactionRepository.
func (m *MockEventDispatcher) SaveTransaction(ctx context.Context, t *models.Transaction) (*dynamodb.PutItemOutput, string, error) {
	args := m.Called(ctx, t)
	return nil, "", args.Error(1)
}

// UpdateFraudTransaction implements db.TransactionRepository.
func (m *MockEventDispatcher) UpdateFraudTransaction(ctx context.Context, phoneNumber string, isFraud bool, status string) (int, error) {
	args := m.Called(ctx, phoneNumber, isFraud, status)
	return 1, args.Error(1)
}

// UpdateTransaction implements db.TransactionRepository.
func (m *MockEventDispatcher) UpdateTransaction(ctx context.Context, accountID string, transactionID string, values *models.Transaction) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, accountID, transactionID, values)
	return nil, args.Error(1)
}

// DispatchFraudUpdateEvent implements events.EventDispatcher.
func (m *MockEventDispatcher) DispatchFraudUpdateEvent(number string, body string) error {
	args := m.Called(number, body)
	return args.Error(0)
}

type MockFraudService struct {
	mock.Mock
}

func (m *MockEventDispatcher) DispatchFraudAlertEvent(txn models.Transaction) error {
	args := m.Called(txn)
	return args.Error(0)
}

func (m *MockFraudService) PredictFraud(ctx context.Context, transactions []models.Transaction) ([]models.Transaction, error) {
	args := m.Called(ctx, transactions)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

type PredictFraudTestSuite struct {
	suite.Suite
	mockEventDispatcher       *MockEventDispatcher
	mockFraudService          *MockFraudService
	mockTransactionRepository *MockTransactionRepository
}

func (suite *PredictFraudTestSuite) SetupTest() {
	suite.mockEventDispatcher = new(MockEventDispatcher)
	suite.mockFraudService = new(MockFraudService)
	suite.mockTransactionRepository = new(MockTransactionRepository)
}

// Fraud Service Tests
func (suite *PredictFraudTestSuite) TestNoFraudDetected() {
	ctx := context.Background()
	// Arrange
	transactions := []models.Transaction{
		{Email: "safeuser@example.com", AccountID: "1", TransactionID: "1"},
		{Email: "anotheruser@example.com", AccountID: "2", TransactionID: "2"},
	}
	suite.mockTransactionRepository.On(
		"UpdateTransaction",
		ctx,
		"1", "1",
		mock.MatchedBy(func(t *models.Transaction) bool {
			return t.Email == "safeuser@example.com" && t.TransactionStatus == "APPROVED"
		}),
	).Return(nil, nil).Once()

	suite.mockTransactionRepository.On(
		"UpdateTransaction",
		ctx,
		"2", "2",
		mock.MatchedBy(func(t *models.Transaction) bool {
			return t.Email == "anotheruser@example.com" && t.TransactionStatus == "APPROVED"
		}),
	).Return(nil, nil).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher, suite.mockTransactionRepository)

	// Act
	_, err := fraudService.PredictFraud(ctx, transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return an error for non-fraud transactions")
}

func (suite *PredictFraudTestSuite) TestFraudDetected() {
	ctx := context.Background()
	// Arrange
	transactions := []models.Transaction{
		{Email: "rshart@wisc.edu", AccountID: "1", TransactionID: "1"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[0]).Return(nil).Once()

	suite.mockEventDispatcher.On("UpdateTransaction", ctx, "1", "1", mock.MatchedBy(func(t *models.Transaction) bool {
		return t.AccountID == "1" && t.TransactionID == "1"
	})).Return(nil, nil).Once()

	fraudService := services.NewFraudService(suite.mockEventDispatcher, suite.mockEventDispatcher)

	// Act
	_, err := fraudService.PredictFraud(ctx, transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return an error when fraud alert is successfully dispatched")
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestFraudDispatchFails() {
	ctx := context.Background()
	// Arrange
	transactions := []models.Transaction{
		{Email: "rshart@wisc.edu", AccountID: "1", TransactionID: "1"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[0]).Return(errors.New("dispatch error")).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher, suite.mockTransactionRepository)

	// Act

	_, err := fraudService.PredictFraud(ctx, transactions)

	// Assert
	assert.Error(suite.T(), err, "Should return an error when fraud alert dispatch fails")
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestConcurrentTransactions() {
	ctx := context.Background()
	// Arrange
	transactions := []models.Transaction{
		{Email: "jalarsen5@wisc.edu", AccountID: "1", TransactionID: "1"},
		{Email: "rshart@wisc.edu", AccountID: "2", TransactionID: "2"},
		{Email: "jpoconnell4@wisc.edu", AccountID: "3", TransactionID: "3"},
	}
	suite.mockTransactionRepository.On(
		"UpdateTransaction",
		ctx,
		"1", "1",
		mock.MatchedBy(func(t *models.Transaction) bool {
			return t.Email == "jalarsen5@wisc.edu" && t.TransactionStatus == "APPROVED"
		}),
	).Return(nil, nil).Once()

	suite.mockTransactionRepository.On(
		"UpdateTransaction",
		ctx,
		"2", "2",
		mock.MatchedBy(func(t *models.Transaction) bool {
			return t.Email == "rshart@wisc.edu" && t.TransactionStatus == "POTENTIAL_FRAUD"
		}),
	).Return(nil, nil).Once()

	suite.mockTransactionRepository.On(
		"UpdateTransaction",
		ctx,
		"3", "3",
		mock.MatchedBy(func(t *models.Transaction) bool {
			return t.Email == "jpoconnell4@wisc.edu" && t.TransactionStatus == "POTENTIAL_FRAUD"
		}),
	).Return(nil, nil).Once()

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[1]).Return(nil).Once()
	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[2]).Return(nil).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher, suite.mockTransactionRepository)

	// Act
	_, err := fraudService.PredictFraud(ctx, transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return error for multiple transactions")
	suite.mockTransactionRepository.AssertExpectations(suite.T())
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

// Fraud Detection Handler Tests

func (suite *PredictFraudTestSuite) TestHandleRequest() {
	// Arrange
	testTxn1 := GetTestTransaction("test@example.com")

	event := events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			{
				EventID:   "1",
				EventName: "INSERT",
				Change: events.DynamoDBStreamRecord{
					NewImage: testTxn1.ToDynamoDBAttributeValueMap(),
				},
			},
		},
	}

	suite.mockFraudService.On("PredictFraud", mock.Anything, []models.Transaction{testTxn1}).Return([]models.Transaction{}, nil).Once()
	handler := handlers.NewFraudHandler(suite.mockFraudService)

	// Act
	err := handler.ProcessFraudEvent(context.TODO(), event)

	// Assert
	assert.Nil(suite.T(), err)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestHandleMultipleTransactionRequest() {
	// Arrange
	testTxn1 := GetTestTransaction("test@example.com")
	testTxn2 := GetTestTransaction("jpoconnell4@wisc.edu")
	testTxn3 := GetTestTransaction("test@example.com")
	shouldSucceed := []models.Transaction{testTxn1, testTxn2}

	event := events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			getDynamoDBEventRecord(testTxn1, "INSERT"),
			getDynamoDBEventRecord(testTxn2, "INSERT"),
			getDynamoDBEventRecord(testTxn3, "MODIFY"),
		},
	}

	suite.mockFraudService.On("PredictFraud", mock.Anything, shouldSucceed).Return([]models.Transaction{}, nil).Once()
	handler := handlers.NewFraudHandler(suite.mockFraudService)

	// Act
	err := handler.ProcessFraudEvent(context.TODO(), event)

	// Assert
	assert.Nil(suite.T(), err)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func TestPredictFraudSuite(t *testing.T) {
	suite.Run(t, new(PredictFraudTestSuite))
}

func getDynamoDBEventRecord(txn models.Transaction, dbEventType string) events.DynamoDBEventRecord {
	return events.DynamoDBEventRecord{
		EventID:   uuid.New().String(),
		EventName: dbEventType,
		Change: events.DynamoDBStreamRecord{
			NewImage: txn.ToDynamoDBAttributeValueMap(),
		},
	}
}
