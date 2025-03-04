package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TransactionRepository is the data access layer for transactions.
type TransactionRepository struct {
	DB *DynamoDBClient
}

// NewTransactionRepository initializes a new repository instance.
func NewTransactionRepository(db *DynamoDBClient) *TransactionRepository {
	return &TransactionRepository{DB: db}
}

// SaveTransaction validates and inserts a new transaction.
func (r *TransactionRepository) SaveTransaction(ctx context.Context, t *models.Transaction) (*dynamodb.PutItemOutput, string, error) {
	// Validate the transaction before saving
	if err := t.ValidateTransaction(); err != nil {
		return nil, "", fmt.Errorf("validation failed: %w", err)
	}

	// Marshal transaction into a DynamoDB-compatible format
	item, err := t.MarshalDynamoDB()
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Insert transaction into DynamoDB
	output, metadata, err := r.DB.PutItem(ctx, item)
	if err != nil {
		return nil, "", err
	}

	// Log metadata for monitoring purposes
	fmt.Printf("Transaction saved: %s | Metadata: %s\n", t.TransactionID, metadata)

	return output, metadata, nil
}

// GetTransaction retrieves a transaction by TransactionID.
func (r *TransactionRepository) GetTransaction(ctx context.Context, transactionID string) (*models.Transaction, error) {
	// Validate input
	if transactionID == "" {
		return nil, errors.New("transactionID cannot be empty")
	}

	// Define the key
	key := map[string]types.AttributeValue{
		"TransactionID": &types.AttributeValueMemberS{Value: transactionID},
	}

	// Fetch transaction from DynamoDB
	item, err := r.DB.GetItem(ctx, key)
	if err != nil {
		return nil, err // Handles "item not found" and other errors
	}

	// Unmarshal into a Transaction struct
	transaction, err := models.UnmarshalDynamoDB(item)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return transaction, nil
}

// UpdateTransaction updates a transaction's fields.
func (r *TransactionRepository) UpdateTransaction(ctx context.Context, transactionID string, values *models.Transaction) (*dynamodb.UpdateItemOutput, error) {
	// Validate transaction ID
	if transactionID == "" {
		return nil, errors.New("transactionID cannot be empty")
	}
	// Convert struct to a map for partial updates
	updates, err := values.TransactionUpdatePayload()
	if err != nil {
		return nil, fmt.Errorf("failed to convert transaction to update map: %w", err)
	}
	// Ensure at least one field is provided for update
	if len(updates) == 0 {
		return nil, errors.New("no fields provided for update")
	}

	// Call DynamoDB update function
	result, err := r.DB.UpdateItem(ctx, transactionID, updates)
	if err != nil {
		return nil, err
	}

	// Log updated values
	fmt.Printf("Transaction updated: %s | UpdatedFields: %v\n", transactionID, result.Attributes)

	return result, nil
}

// DeleteTransaction removes a transaction by TransactionID.
func (r *TransactionRepository) DeleteTransaction(ctx context.Context, transactionID string) error {
	// Validate input
	if transactionID == "" {
		return errors.New("transactionID cannot be empty")
	}

	// Define the primary key
	key := map[string]types.AttributeValue{
		"TransactionID": &types.AttributeValueMemberS{Value: transactionID},
	}

	// Call DynamoDB delete function
	_, err := r.DB.DeleteItem(ctx, key)
	if err != nil {
		return err
	}

	fmt.Printf("Transaction deleted: %s\n", transactionID)
	return nil
}
