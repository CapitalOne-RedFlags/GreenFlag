package test

import (
	"context"
	"errors"
	"testing"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/handlers"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-lambda-go/events"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockEventDispatcher struct {
	mock.Mock
}

type MockFraudService struct {
	mock.Mock
}

func (m *MockEventDispatcher) DispatchFraudAlertEvent(txn models.Transaction) error {
	args := m.Called(txn)
	return args.Error(0)
}

func (m *MockFraudService) PredictFraud(transactions []models.Transaction) ([]models.Transaction, []models.Transaction, error) {
	args := m.Called(transactions)
	return args.Get(0).([]models.Transaction), args.Get(1).([]models.Transaction), args.Error(2)
}

type PredictFraudTestSuite struct {
	suite.Suite
	mockEventDispatcher *MockEventDispatcher
	mockFraudService    *MockFraudService
}

func (suite *PredictFraudTestSuite) SetupTest() {
	suite.mockEventDispatcher = new(MockEventDispatcher)
	suite.mockFraudService = new(MockFraudService)
}

// Fraud Service Tests
func (suite *PredictFraudTestSuite) TestNoFraudDetected() {
	// Arrange
	transactions := []models.Transaction{
		{Email: "safeuser@example.com"},
		{Email: "anotheruser@example.com"},
	}

	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	_, failedTransactions, err := fraudService.PredictFraud(transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return an error for non-fraud transactions")
	assert.Empty(suite.T(), failedTransactions)
}

func (suite *PredictFraudTestSuite) TestFraudDetected() {
	// Arrange
	transactions := []models.Transaction{
		{Email: "rshart@wisc.edu"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[0]).Return(nil).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	_, failedTransactions, err := fraudService.PredictFraud(transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return an error when fraud alert is successfully dispatched")
	assert.Empty(suite.T(), failedTransactions)
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestFraudDispatchFails() {
	// Arrange
	transactions := []models.Transaction{
		{Email: "rshart@wisc.edu"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[0]).Return(errors.New("dispatch error")).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	_, failedTransactions, err := fraudService.PredictFraud(transactions)

	// Assert
	assert.Error(suite.T(), err, "Should return an error when fraud alert dispatch fails")
	assert.Len(suite.T(), failedTransactions, 1)
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestConcurrentTransactions() {
	// Arrange
	transactions := []models.Transaction{
		{Email: "jalarsen5@wisc.edu"},
		{Email: "rshart@wisc.edu"},
		{Email: "jpoconnell4@wisc.edu"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[1]).Return(nil).Once()
	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[2]).Return(nil).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	_, failedTransactions, err := fraudService.PredictFraud(transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return error for multiple transactions")
	assert.Empty(suite.T(), failedTransactions)
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

	suite.mockFraudService.On("PredictFraud", []models.Transaction{testTxn1}).Return([]models.Transaction{}, []models.Transaction{}, nil).Once()
	handler := handlers.NewFraudHandler(suite.mockFraudService)

	// Act
	batchResult, err := handler.ProcessFraudEvent(context.TODO(), event)

	// Assert
	assert.Nil(suite.T(), err)
	assert.Empty(suite.T(), batchResult.BatchItemFailures)
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

	suite.mockFraudService.On("PredictFraud", shouldSucceed).Return([]models.Transaction{}, []models.Transaction{}, nil).Once()
	handler := handlers.NewFraudHandler(suite.mockFraudService)

	// Act
	failedTransactions, err := handler.ProcessFraudEvent(context.TODO(), event)

	// Assert
	assert.Nil(suite.T(), err)
	assert.Empty(suite.T(), failedTransactions)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestFraudHandler_PartialBatchFailure() {
	// Arrange
	testTxn1 := GetTestTransaction("test@example.com")
	testTxn2 := GetTestTransaction("jpoconnell4@wisc.edu")
	testTxn3 := GetTestTransaction("test@example.com")
	serviceArgs := []models.Transaction{testTxn1, testTxn2, testTxn3}

	eventRecord1 := getDynamoDBEventRecord(testTxn1, "INSERT")
	eventRecord2 := getDynamoDBEventRecord(testTxn2, "INSERT")
	eventRecord3 := getDynamoDBEventRecord(testTxn3, "INSERT")

	event := events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			eventRecord1,
			eventRecord2,
			eventRecord3,
		},
	}

	suite.mockFraudService.On("PredictFraud", serviceArgs).Return([]models.Transaction{}, []models.Transaction{testTxn1, testTxn3}, errors.New("Test")).Once()
	expectedRIDs := []string{eventRecord1.Change.SequenceNumber, eventRecord3.Change.SequenceNumber}
	handler := handlers.NewFraudHandler(suite.mockFraudService)

	// Act
	batchResult, err := handler.ProcessFraudEvent(context.TODO(), event)

	// Assert
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), batchResult)
	assert.Len(suite.T(), batchResult.BatchItemFailures, 2)
	assert.ElementsMatch(suite.T(), batchResult.GetRids(), expectedRIDs)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestRetryFraudPipeline() {
	// Arrange
	testTxn1 := GetTestTransaction("test@example.com")
	testTxn2 := GetTestTransaction("jpoconnell4@wisc.edu")
	testTxn3 := GetTestTransaction("test@example.com")
	serviceArgs := []models.Transaction{testTxn1, testTxn2, testTxn3}

	event := events.SQSEvent{
		Records: []events.SQSMessage{
			getSQSEventRecord(testTxn1),
			getSQSEventRecord(testTxn2),
			getSQSEventRecord(testTxn3),
		},
	}

	suite.mockFraudService.On("PredictFraud", serviceArgs).Return([]models.Transaction{}, []models.Transaction{}, nil).Once()
	handler := handlers.NewFraudRetryHandler(suite.mockFraudService)

	// Act
	failedTransactions, err := handler.ProcessDLQFraudEvent(context.TODO(), event)

	// Assert
	assert.Nil(suite.T(), err)
	assert.Empty(suite.T(), failedTransactions)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestRetryFraudPipeline_PartialBatchFailure() {
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

	suite.mockFraudService.On("PredictFraud", serviceArgs).Return([]models.Transaction{}, []models.Transaction{testTxn2, testTxn3}, errors.New("Test")).Once()
	expectedRIDs := []string{eventRecord2.MessageId, eventRecord3.MessageId}
	handler := handlers.NewFraudRetryHandler(suite.mockFraudService)

	// Act
	batchResult, err := handler.ProcessDLQFraudEvent(context.TODO(), event)

	// Assert
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), batchResult)
	assert.Len(suite.T(), batchResult.BatchItemFailures, 2)
	assert.ElementsMatch(suite.T(), batchResult.GetRids(), expectedRIDs)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestFraudRetryHandler_ShouldPartialFail_WithInvalidTransactionBody() {
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

	suite.mockFraudService.On(
		"PredictFraud",
		serviceArgs,
	).Return(
		[]models.Transaction{},
		[]models.Transaction{},
		nil,
	).Once()

	expectedRIDs := []string{eventRecord3.MessageId}
	handler := handlers.NewFraudRetryHandler(suite.mockFraudService)

	// Act
	batchResult, err := handler.ProcessDLQFraudEvent(context.TODO(), event)

	// Assert
	assert.NotNil(suite.T(), err)
	assert.NotNil(suite.T(), batchResult)
	assert.Len(suite.T(), batchResult.BatchItemFailures, 1)
	assert.ElementsMatch(suite.T(), batchResult.GetRids(), expectedRIDs)
	suite.mockFraudService.AssertExpectations(suite.T())
}

func TestPredictFraudSuite(t *testing.T) {
	suite.Run(t, new(PredictFraudTestSuite))
}
