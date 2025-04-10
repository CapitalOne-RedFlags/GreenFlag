package models

import (
	"encoding/json"
)

type TwilioMessage struct {
	MessagingServiceSid string `json:"MessagingServiceSid"`
	ApiVersion          string `json:"ApiVersion"`
	SmsSid              string `json:"SmsSid"`
	SmsStatus           string `json:"SmsStatus"`
	SmsMessageSid       string `json:"SmsMessageSid"`
	NumSegments         string `json:"NumSegments"`
	ToState             string `json:"ToState"`
	From                string `json:"From"`
	MessageSid          string `json:"MessageSid"`
	AccountSid          string `json:"AccountSid"`
	ToCity              string `json:"ToCity"`
	FromCountry         string `json:"FromCountry"`
	ToZip               string `json:"ToZip"`
	FromCity            string `json:"FromCity"`
	To                  string `json:"To"`
	FromZip             string `json:"FromZip"`
	ToCountry           string `json:"ToCountry"`
	Body                string `json:"Body"`
	NumMedia            string `json:"NumMedia"`
	FromState           string `json:"FromState"`
}

func UnmarshalResponseSQS(message string) (*TwilioMessage, error) {
	var result TwilioMessage
	err := json.Unmarshal([]byte(message), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil

}
