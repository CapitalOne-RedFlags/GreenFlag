package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	internalConfig "github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/messaging"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
)

// ProcessingState tracks which transactions have been processed
type ProcessingState struct {
	LastProcessedIndex int       `json:"lastProcessedIndex"`
	LastRunTime        time.Time `json:"lastRunTime"`
}

func main() {
	// Load application configuration
	internalConfig.InitializeConfig()

	// Get batch size from environment variable (set in template.yaml)
	batchSize, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil || batchSize <= 0 {
		batchSize = 10 // Default batch size if not specified or invalid
		log.Printf("Using default batch size: %d", batchSize)
	} else {
		log.Printf("Using configured batch size: %d", batchSize)
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create SQS client
	sqsClient := sqs.NewFromConfig(cfg)
	sqsHandler := messaging.NewSQSHandler(sqsClient, internalConfig.SQSConfig.QueueURL)

	// CSV file path from environment variable or default
	csvFilePath := os.Getenv("CSV_FILE_PATH")
	if csvFilePath == "" {
		csvFilePath = "bank_transactions_data.csv" // Default
	}
	log.Printf("Using CSV file: %s", csvFilePath)
	
	// State file path
	statePath := filepath.Join(filepath.Dir(csvFilePath), "."+filepath.Base(csvFilePath)+".state")
	
	// Load or create processing state
	state, err := loadOrCreateState(statePath)
	if err != nil {
		log.Fatalf("Failed to load or create state: %v", err)
	}

	// Open CSV file
	file, err := os.Open(csvFilePath)
	if err != nil {
		log.Fatalf("Unable to open CSV file: %v", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Error reading CSV header: %v", err)
	}

	// Find column indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[col] = i
	}

	// Skip to last processed index
	currentIndex := 0
	for currentIndex < state.LastProcessedIndex {
		_, err := reader.Read()
		if err == io.EOF {
			log.Printf("Reached end of file while skipping to last processed index")
			return
		}
		if err != nil {
			log.Fatalf("Error skipping records: %v", err)
		}
		currentIndex++
	}

	// Process transactions in batches
	var wg sync.WaitGroup
	var batchCount int
	var totalSent int
	var currentBatch []models.Transaction
	
	// Read and process records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break // End of file
		}
		if err != nil {
			log.Printf("Error reading record: %v", err)
			continue
		}
		
		currentIndex++
		
		// Parse CSV record into transaction
		transaction, err := parseTransaction(record, colMap)
		if err != nil {
			log.Printf("Error parsing transaction at index %d: %v", currentIndex, err)
			continue
		}
		
		// Add to current batch
		currentBatch = append(currentBatch, transaction)
		
		// Process just one transaction and then break
		processBatch(context.TODO(), &wg, sqsHandler, currentBatch, currentIndex-len(currentBatch)+1)
		
		// Update state after successful batch processing
		state.LastProcessedIndex = currentIndex
		if err := saveState(state, statePath); err != nil {
			log.Printf("Error saving state: %v", err)
		}
		
		batchCount++
		totalSent += len(currentBatch)
		log.Printf("Queued batch %d with %d transactions (up to index %d)", 
			batchCount, len(currentBatch), currentIndex)
		
		// Break after processing one transaction
		break
	}
	
	// Wait for all batches to complete
	wg.Wait()
	
	// Update last run time
	state.LastRunTime = time.Now()
	if err := saveState(state, statePath); err != nil {
		log.Printf("Error saving final state: %v", err)
	}
	
	log.Printf("Processing complete. Sent %d transactions in %d batches", totalSent, batchCount)
}

// loadOrCreateState loads the processing state or creates a new one
func loadOrCreateState(statePath string) (*ProcessingState, error) {
	// Try to load existing state
	state := &ProcessingState{
		LastProcessedIndex: 0,
		LastRunTime:        time.Time{},
	}
	
	// Check if state file exists
	if _, err := os.Stat(statePath); err == nil {
		// State exists, load it
		stateFile, err := os.Open(statePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open state file: %w", err)
		}
		defer stateFile.Close()
		
		if err := json.NewDecoder(stateFile).Decode(state); err != nil {
			return nil, fmt.Errorf("failed to decode state: %w", err)
		}
		
		return state, nil
	}
	
	// State doesn't exist, return new state
	return state, nil
}

// saveState saves the processing state to disk
func saveState(state *ProcessingState, statePath string) error {
	// Create temp file
	tempFile, err := os.CreateTemp(filepath.Dir(statePath), ".tmp-state")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	
	// Write state to temp file
	if err := json.NewEncoder(tempFile).Encode(state); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode state: %w", err)
	}
	
	tempFile.Close()
	
	// Replace state file with temp file
	if err := os.Rename(tempPath, statePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace state file: %w", err)
	}
	
	return nil
}

// parseTransaction converts a CSV record to a Transaction using column map
func parseTransaction(record []string, colMap map[string]int) (models.Transaction, error) {
	// Parse numeric fields
	customerAge, _ := strconv.Atoi(record[colMap["customerAge"]])
	transactionDuration, _ := strconv.Atoi(record[colMap["transactionDuration"]])
	loginAttempts, _ := strconv.Atoi(record[colMap["loginAttempts"]])
	accountBalance, _ := strconv.ParseFloat(record[colMap["accountBalance"]], 64)
	amount, _ := strconv.ParseFloat(record[colMap["transactionAmount"]], 64)

	transaction := models.Transaction{
		TransactionID:          record[colMap["transactionId"]],
		AccountID:              record[colMap["accountId"]],
		TransactionAmount:      amount,
		TransactionDate:        record[colMap["transactionDate"]],
		TransactionType:        record[colMap["transactionType"]],
		Location:               record[colMap["location"]],
		DeviceID:               record[colMap["deviceId"]],
		IPAddress:              record[colMap["ipAddress"]],
		MerchantID:             record[colMap["merchantId"]],
		Channel:                record[colMap["channel"]],
		CustomerAge:            customerAge,
		CustomerOccupation:     record[colMap["customerOccupation"]],
		TransactionDuration:    transactionDuration,
		LoginAttempts:          loginAttempts,
		AccountBalance:         accountBalance,
		PreviousTransactionDate: record[colMap["previousTransactionDate"]],
		PhoneNumber:            record[colMap["phoneNumber"]],
		Email:                  record[colMap["email"]],
		TransactionStatus:      record[colMap["transactionStatus"]],
	}

	return transaction, nil
}

// processBatch sends a batch of transactions to SQS
func processBatch(ctx context.Context, wg *sync.WaitGroup, sqsHandler *messaging.SQSHandler, 
                 batch []models.Transaction, startIndex int) {
	wg.Add(1)
	go func(b []models.Transaction) {
		defer wg.Done()
		
		startTime := time.Now()
		successCount := 0
		
		for _, transaction := range b {
			txCopy := transaction // Create a copy to avoid race conditions
			err := sqsHandler.SendTransaction(ctx, &txCopy)
			
			if err != nil {
				log.Printf("Error sending transaction %s: %v", transaction.TransactionID, err)
			} else {
				successCount++
				fmt.Printf("Processed transaction %s\n", transaction.TransactionID)
			}
		}
		
		duration := time.Since(startTime)
		log.Printf("Batch completed: %d/%d successful in %v", successCount, len(b), duration)
	}(batch)
}

// Helper functions for parsing and formatting
func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func formatDate(s string) string {
	// Parse the date from the CSV format (e.g., "4:29:00 PM")
	// and convert it to ISO 8601 format (e.g., "2025-03-11T10:12:34Z")
	// This will depend on your actual date format in the CSV
	
	// Example implementation (adjust based on your actual date format):
	t, err := time.Parse("3:04:05 PM", s)
	if err != nil {
		// If parsing fails, use current time
		t = time.Now()
	}
	
	// Set the date to today (since your CSV might only have time)
	now := time.Now()
	t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	
	return t.Format(time.RFC3339)
} 