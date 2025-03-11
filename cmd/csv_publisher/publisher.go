package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
)

type Transaction struct {
	TransactionID           string  `json:"transactionId"`
	AccountID              string  `json:"accountId"`
	TransactionAmount      float64 `json:"transactionAmount"`
	TransactionDate        string  `json:"transactionDate"`
	TransactionType        string  `json:"transactionType"`
	Location              string  `json:"location"`
	DeviceID              string  `json:"deviceId"`
	IPAddress             string  `json:"ipAddress"`
	MerchantID            string  `json:"merchantId"`
	Channel               string  `json:"channel"`
	CustomerAge           int     `json:"customerAge"`
	CustomerOccupation    string  `json:"customerOccupation"`
	TransactionDuration   int     `json:"transactionDuration"`
	LoginAttempts         int     `json:"loginAttempts"`
	AccountBalance        float64 `json:"accountBalance"`
	PreviousTransactionDate string `json:"previousTransactionDate"`
	PhoneNumber           string  `json:"phoneNumber"`
	Email                 string  `json:"email"`
	TransactionStatus     string  `json:"transactionStatus"`
}

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create SQS client
	sqsClient := sqs.NewFromConfig(cfg)
	sqsHandler := messaging.NewSQSHandler(sqsClient, config.AppConfig.SQSConfig.QueueURL)

	// Open CSV file
	file, err := os.Open("bank_transactions_data.csv")
	if err != nil {
		log.Fatalf("Unable to open CSV file: %v", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		log.Fatalf("Error reading CSV header: %v", err)
	}

	// Read and publish each transaction
	for {
		record, err := reader.Read()
		if err != nil {
			break // End of file
		}

		// Parse CSV record into transaction
		customerAge, _ := strconv.Atoi(record[10])
		transactionDuration, _ := strconv.Atoi(record[12])
		loginAttempts, _ := strconv.Atoi(record[13])
		accountBalance, _ := strconv.ParseFloat(record[14], 64)
		amount, _ := strconv.ParseFloat(record[2], 64)

		transaction := Transaction{
			TransactionID:           record[0],
			AccountID:              record[1],
			TransactionAmount:      amount,
			TransactionDate:        record[3],
			TransactionType:        record[4],
			Location:              record[5],
			DeviceID:              record[6],
			IPAddress:             record[7],
			MerchantID:            record[8],
			Channel:               record[9],
			CustomerAge:           customerAge,
			CustomerOccupation:    record[11],
			TransactionDuration:   transactionDuration,
			LoginAttempts:         loginAttempts,
			AccountBalance:        accountBalance,
			PreviousTransactionDate: record[15],
			PhoneNumber:           record[16],
			Email:                 record[17],
			TransactionStatus:     record[18],
		}

		// Convert transaction to JSON
		jsonData, err := json.Marshal(transaction)
		if err != nil {
			log.Printf("Error marshaling transaction %s: %v", transaction.TransactionID, err)
			continue
		}

		// Send to SQS
		err = sqsHandler.SendTransaction(context.TODO(), &transaction)
		if err != nil {
			log.Printf("Error sending transaction %s: %v", transaction.TransactionID, err)
			continue
		}

		fmt.Printf("Queued transaction %s\n", transaction.TransactionID)
	}
} 