package test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

func (m *MockFraudService) PredictFraud(transactions []models.Transaction) error {
	args := m.Called(transactions)
	return args.Error(0)
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
	err := fraudService.PredictFraud(transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return an error for non-fraud transactions")
}

func (suite *PredictFraudTestSuite) TestFraudDetected() {
	// Arrange
	transactions := []models.Transaction{
		{Email: "rshart@wisc.edu"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[0]).Return(nil).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	err := fraudService.PredictFraud(transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return an error when fraud alert is successfully dispatched")
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestFraudDispatchFails() {
	transactions := []models.Transaction{
		{Email: "rshart@wisc.edu"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[0]).Return(errors.New("dispatch error")).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	err := fraudService.PredictFraud(transactions)

	// Assert
	assert.Error(suite.T(), err, "Should return an error when fraud alert dispatch fails")
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

func (suite *PredictFraudTestSuite) TestConcurrentTransactions() {
	// Arrange
	transactions := []models.Transaction{
		{Email: "jalarsen5@wisc.edu"},
		{Email: "rshart@wisc.edu"},
		{Email: "jpconnell4@wisc.eud"},
	}

	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[1]).Return(nil).Once()
	suite.mockEventDispatcher.On("DispatchFraudAlertEvent", transactions[2]).Return(nil).Once()
	fraudService := services.NewFraudService(suite.mockEventDispatcher)

	// Act
	err := fraudService.PredictFraud(transactions)

	// Assert
	assert.NoError(suite.T(), err, "Should not return error for multiple transactions")
	suite.mockEventDispatcher.AssertExpectations(suite.T())
}

// Fraud Detection Handler Tests

func (suite *PredictFraudTestSuite) TestHandleRequest() {
	// Arrange
	testTxn := GetTestTransaction("test@example.com")

	event := events.DynamoDBEvent{
		Records: []events.DynamoDBEventRecord{
			{
				EventID:   "1",
				EventName: "INSERT",
				Change: events.DynamoDBStreamRecord{
					NewImage: toDynamoDBAttributeValues(&testTxn),
				},
			},
		},
	}

	suite.mockFraudService.On("PredictFraud", []models.Transaction{testTxn}).Return(nil).Once()
	handler := handlers.NewFraudHandler(suite.mockFraudService)

	// Act
	err := handler.ProcessFraudEvent(context.TODO(), event)

	// Assert
	assert.Nil(suite.T(), err)
}

// TODO: Move to transaction model. May need to add support for more types if DB grows
func toDynamoDBAttributeValues(txn *models.Transaction) map[string]events.DynamoDBAttributeValue {
	val := reflect.ValueOf(txn)
	typ := reflect.TypeOf(txn)

	avMap := make(map[string]events.DynamoDBAttributeValue)

	if val.Kind() == reflect.Struct {
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			fieldName := typ.Field(i).Name
			fmt.Printf("Field: %s, Value: %v\n", fieldName, field.Interface())

			switch field.Kind() {
			case reflect.String:
				avMap[fieldName] = events.NewStringAttribute(field.String())
			case reflect.Float64:
				avMap[fieldName] = events.NewNumberAttribute(field.String())
			case reflect.Int:
				avMap[fieldName] = events.NewNumberAttribute(field.String())
			default:
			}
		}
	}

	return avMap
}

func TestPredictFraudSuite(t *testing.T) {
	suite.Run(t, new(PredictFraudTestSuite))
}
