package test

import (
	"context"
	"log"
	"sync"
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
	topicArns    []string
	arnLock      sync.Mutex
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

	s.snsMessenger = messaging.NewGfSNSMessenger(client)
}

func (s *SNSMessagingTestSuite) TestPublishEmailAlert() {
	// Arrange
	txn := GetTestTransaction("jalarsen5@wisc.edu")

	// Act
	output, topicArn, err := s.snsMessenger.PublishEmailAlert(txn)

	// Assert
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), topicArn)

	addTopicArnToTeardown(s, topicArn)

	assert.NotNil(s.T(), output)
}

func (s *SNSMessagingTestSuite) TearDownSuite() {
	s.arnLock.Lock()

	for _, topicArn := range s.topicArns {
		deleteTopicInput := &sns.DeleteTopicInput{
			TopicArn: &topicArn,
		}

		_, err := s.snsMessenger.Client.DeleteTopic(context.TODO(), deleteTopicInput)
		if err != nil {
			log.Fatalf("Failed to delete SNS Topic: %s\n", err)
		}
	}

	s.arnLock.Unlock()
}

// Call when a topic arn is created in a test method
func addTopicArnToTeardown(s *SNSMessagingTestSuite, topicArn *string) {
	s.arnLock.Lock()
	defer s.arnLock.Unlock()
	s.topicArns = append(s.topicArns, *topicArn)
}
