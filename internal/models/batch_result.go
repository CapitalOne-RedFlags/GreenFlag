package models

type BatchResult struct {
	BatchItemFailures []BatchItemFailure `json:"BatchItemFailures"`
}

type BatchItemFailure struct {
	ItemIdentifier string `json:"ItemIdentifier"`
}
