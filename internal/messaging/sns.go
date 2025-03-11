package messaging

import (
	"context"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SNSMessenger interface {
	PublishEmailAlert(transaction models.Transaction) (*sns.PublishOutput, *string, error)
}

type GfSNSMessenger struct {
	Client *sns.Client
}

func NewGfSNSMessenger(snsClient *sns.Client) *GfSNSMessenger {
	return &GfSNSMessenger{
		Client: snsClient,
	}
}

func (messenger *GfSNSMessenger) PublishEmailAlert(transaction models.Transaction) (*sns.PublishOutput, *string, error) {
	topicName := fmt.Sprintf("%s_%s", config.SNSMessengerConfig.TopicNamePrefix, transaction.AccountID)

	topicArn, err := messenger.createTopic(topicName)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create SNS topic: %s\n", err)
	}

	_, err = messenger.subscribeToSNSTopic("email", transaction.Email, topicArn)
	if err != nil {
		return nil, &topicArn, fmt.Errorf("Failed to subscribe %s SNS topic: %s\n", transaction.Email, err)
	}

	subject, message := transaction.GetFraudEmailContent()

	input := &sns.PublishInput{
		Message:  aws.String(message),
		Subject:  aws.String(subject),
		TopicArn: aws.String(topicArn),
	}

	publishOutput, err := messenger.Client.Publish(context.TODO(), input)
	if err != nil {
		return nil, &topicArn, fmt.Errorf("Failed to send SNS email: %v", err)
	}

	return publishOutput, &topicArn, nil
}

func (messenger *GfSNSMessenger) createTopic(topicName string) (string, error) {
	input := &sns.CreateTopicInput{
		Name: aws.String(topicName),
	}

	result, err := messenger.Client.CreateTopic(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to create SNS topic: %v", err)
	}

	return *result.TopicArn, nil
}

func (messenger *GfSNSMessenger) subscribeToSNSTopic(protocol string, endpoint string, topicArn string) (*sns.SubscribeOutput, error) {
	input := &sns.SubscribeInput{
		TopicArn: aws.String(topicArn),
		Protocol: aws.String(protocol), // "email", "sms", "lambda", etc.
		Endpoint: aws.String(endpoint), // email address or phone number
	}

	subscribeOutput, err := messenger.Client.Subscribe(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to SNS topic: %v", err)
	}

	return subscribeOutput, nil
}
