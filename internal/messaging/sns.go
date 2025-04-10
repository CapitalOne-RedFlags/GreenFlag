package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type SNSMessenger interface {
	SendEmailAlert(transaction models.Transaction) (*sns.PublishOutput, error)
	SendTextAlert(transaction models.Transaction) error
}

type GfSNSMessenger struct {
	Client         *sns.Client
	TopicName      string
	TopicArn       string
	TwilioUsername string
	TwilioPassword string
}

func NewGfSNSMessenger(snsClient *sns.Client, topicName string, topicArn string, twilioUsernmae string, twilioPassword string) *GfSNSMessenger {
	return &GfSNSMessenger{
		Client:         snsClient,
		TopicName:      topicName,
		TopicArn:       topicArn,
		TwilioUsername: twilioUsernmae,
		TwilioPassword: twilioPassword,
	}
}

func CreateTopic(client *sns.Client, topicName string) (string, error) {
	input := &sns.CreateTopicInput{
		Name: aws.String(topicName),
	}

	result, err := client.CreateTopic(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to create SNS topic: %v", err)
	}

	return *result.TopicArn, nil
}

func (messenger *GfSNSMessenger) SendEmailAlert(transaction models.Transaction) (*sns.PublishOutput, error) {
	_, err := messenger.SubscribeToSNSTopic("email", transaction.Email, transaction.AccountID)
	if err != nil {
		return nil, fmt.Errorf("Failed to subscribe %s SNS topic: %s\n", transaction.Email, err)
	}

	publishOutput, err := messenger.PublishEmailMessage(transaction)
	if err != nil {
		return nil, fmt.Errorf("Failed to publish transaction with id %s SNS topic: %s\n", transaction.TransactionID, err)
	}

	return publishOutput, nil
}

func (messenger *GfSNSMessenger) PublishEmailMessage(transaction models.Transaction) (*sns.PublishOutput, error) {
	subject, message := transaction.GetFraudEmailContent()
	messageAttributes := GetMessageAttributes(transaction)

	input := &sns.PublishInput{
		Message:           aws.String(message),
		Subject:           aws.String(subject),
		TopicArn:          aws.String(messenger.TopicArn),
		MessageAttributes: messageAttributes,
	}

	publishOutput, err := messenger.Client.Publish(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("Failed to send SNS email: %v", err)
	}

	return publishOutput, nil
}

func (messenger *GfSNSMessenger) SubscribeToSNSTopic(protocol string, endpoint string, accountId string) (*sns.SubscribeOutput, error) {
	// filterPolicy, err := GetFilterPolicy(accountId)
	// if err != nil {
	// 	return nil, fmt.Errorf("Failed to get filter policy: %s\n", err)
	// }

	input := &sns.SubscribeInput{
		TopicArn: aws.String(messenger.TopicArn),
		Protocol: aws.String(protocol), // "email", "sms", "lambda", etc.
		Endpoint: aws.String(endpoint), // email address or phone number
		// Attributes: map[string]string{
		// 	"FilterPolicy": *filterPolicy,
		// },
		ReturnSubscriptionArn: true,
	}

	subscribeOutput, err := messenger.Client.Subscribe(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to SNS topic: %v", err)
	}

	// _, err = messenger.Client.SetSubscriptionAttributes(context.TODO(), &sns.SetSubscriptionAttributesInput{
	// 	SubscriptionArn: aws.String(*subscribeOutput.SubscriptionArn),
	// 	AttributeName:   aws.String("FilterPolicy"),
	// 	AttributeValue:  aws.String(*filterPolicy),
	// })

	if err != nil {
		return nil, fmt.Errorf("Failed to set subscription attributes: %s", err)
	}

	return subscribeOutput, nil
}

func GetMessageAttributes(transaction models.Transaction) map[string]types.MessageAttributeValue {
	return map[string]types.MessageAttributeValue{
		"AccountID": NewMessageAttributeValue("String", transaction.AccountID),
	}
}

func GetFilterPolicy(accountID string) (*string, error) {
	filterPolicy := map[string][]string{
		"AccountID": {accountID},
	}

	policy, err := json.Marshal(filterPolicy)
	if err != nil {
		return nil, fmt.Errorf("Failed to get message filter policy: %s\n", err)
	}

	policyString := string(policy)

	return &policyString, nil
}

func NewMessageAttributeValue(dataType string, stringValue string) types.MessageAttributeValue {
	return types.MessageAttributeValue{
		DataType:    aws.String(dataType),
		StringValue: aws.String(stringValue),
	}
}

func (messenger *GfSNSMessenger) SendTextAlert(transaction models.Transaction) error {

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: messenger.TwilioUsername,
		Password: messenger.TwilioPassword,
	})

	params := &api.CreateMessageParams{}
	_, body := transaction.GetFraudEmailContent()
	params.SetBody(body)

	//ASSIGNED TWILIO PHONE NUMBER
	params.SetFrom("+18333981458")
	number := transaction.PhoneNumber
	params.SetTo(number)

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}
