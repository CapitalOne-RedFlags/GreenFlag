package test

import (
	"context"
	"log"
	"testing"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SNSMessagingTestSuite struct {
	suite.Suite
	snsMessenger *messaging.GfSNSMessenger
	ctx          context.Context
	topicArn     string
}

func TestSNSMessagingSuite(t *testing.T) {
	suite.Run(t, new(SNSMessagingTestSuite))
}

func (s *SNSMessagingTestSuite) SetupSuite() {
	config.InitializeConfig()
	s.ctx = context.Background()

	awsConfig, err := config.LoadAWSConfig(s.ctx)
	if err != nil {
		log.Fatalf("Failed to load AWS Config: %s\n", err)
	}

	client := sns.NewFromConfig(awsConfig.Config)
	if client == nil {
		log.Fatal("Failed to create SNS Client!\n", err)
	}

	topicName := config.SNSMessengerConfig.TopicName

	topicArn, err := messaging.CreateTopic(client, topicName)
	if err != nil {
		log.Fatalf("Failed to create SNS Topic: %s\n", err)
	}
	s.topicArn = topicArn

	s.snsMessenger = messaging.NewGfSNSMessenger(client, config.SNSMessengerConfig.TopicName, topicArn)
}

func (s *SNSMessagingTestSuite) TestSendEmailAlert() {
	// Arrange
	txn := GetTestTransaction("c1redflagstest@gmail.com")

	// Act
	output, topicArn, err := s.snsMessenger.SendEmailAlert(txn)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), topicArn)
	assert.NotNil(s.T(), output)
}

func (s *SNSMessagingTestSuite) TearDownSuite() {
	input := &sns.DeleteTopicInput{
		TopicArn: &s.topicArn,
	}

	_, err := s.snsMessenger.Client.DeleteTopic(context.TODO(), input)
	if err != nil {
		log.Fatalf("Failed to delete topic: %s\n", err)
	}
}
