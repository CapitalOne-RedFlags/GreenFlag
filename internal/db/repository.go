package db

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/middleware"
	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TransactionRepository is the data access layer for transactions.
type TransactionRepository interface {
	SaveTransaction(ctx context.Context, t *models.Transaction) (*dynamodb.PutItemOutput, string, error)
	GetTransaction(ctx context.Context, accountID, transactionID string) (*models.Transaction, error)
	GetFraudTransaction(ctx context.Context, phoneNumber string) ([]models.Transaction, error)
	UpdateTransaction(ctx context.Context, accountID, transactionID string, values *models.Transaction) (*dynamodb.UpdateItemOutput, error)
	UpdateFraudTransaction(ctx context.Context, phoneNumber string, isFraud bool) error
	DeleteTransaction(ctx context.Context, accountID, transactionID string) error
}

// Implementation of the Interface
type DynamoTransactionRepository struct {
	DB *DynamoDBClient
}

// NewTransactionRepository initializes a new repository instance.
func NewTransactionRepository(db *DynamoDBClient) TransactionRepository {
	return &DynamoTransactionRepository{DB: db}
}

// SaveTransaction validates and inserts a new transaction.
func (r *DynamoTransactionRepository) SaveTransaction(ctx context.Context, t *models.Transaction) (*dynamodb.PutItemOutput, string, error) {
	if err := t.ValidateTransaction(); err != nil {
		return nil, "", fmt.Errorf("validation failed: %w", err)
	}

	item, err := t.MarshalDynamoDB()
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal transaction: %w", err)
	}

	output, metadata, err := r.DB.PutItem(ctx, item)
	if err != nil {
		return nil, "", err
	}

	fmt.Printf("Transaction saved: %s | Metadata: %s\n", t.TransactionID, metadata)
	return output, metadata, nil
}

func (r *DynamoTransactionRepository) GetFraudTransaction(ctx context.Context, phoneNumber string) ([]models.Transaction, error) {
	var transactions []models.Transaction
	keyEx := expression.Key("PhoneNumber").Equal(expression.Value(phoneNumber))
	filterEx := expression.Name("TransactionStatus").Equal((expression.Value("POTENTIAL_FRAUD")))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).WithFilter(filterEx).Build()
	if err != nil {
		fmt.Printf("Couldn't build expression for query. Here's why: %v\n", err)
		return nil, err
	} else {
		queryPaginator := dynamodb.NewQueryPaginator(r.DB.Client, &dynamodb.QueryInput{
			TableName:                 aws.String(r.DB.TableName),
			IndexName:                 aws.String("PhoneNumberIndex"),
			KeyConditionExpression:    expr.KeyCondition(),
			FilterExpression:          expr.Filter(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
		})
		for queryPaginator.HasMorePages() {
			response, err := queryPaginator.NextPage(ctx)
			if err != nil {
				fmt.Printf("Error getting transactions: %s", err)
				return nil, err

			} else {
				var transactionsPage []models.Transaction
				err = attributevalue.UnmarshalListOfMaps(response.Items, &transactionsPage)
				if err != nil {
					fmt.Printf("Couldn't unmarshal query response. Here's why: %v\n", err)
					return nil, err
				} else {
					transactions = append(transactions, transactionsPage...)
				}
			}
		}
	}
	return transactions, nil
}

// GetTransaction retrieves a transaction by AccountID and TransactionID
func (r *DynamoTransactionRepository) GetTransaction(ctx context.Context, accountID, transactionID string) (*models.Transaction, error) {
	// Validate input using config keys
	if accountID == "" {
		return nil, fmt.Errorf("%s cannot be empty", config.DBConfig.Keys.PartitionKey)
	}
	if transactionID == "" {
		return nil, fmt.Errorf("%s cannot be empty", config.DBConfig.Keys.SortKey)
	}

	// Define key using config
	key := map[string]types.AttributeValue{
		config.DBConfig.Keys.PartitionKey: &types.AttributeValueMemberS{Value: accountID},
		config.DBConfig.Keys.SortKey:      &types.AttributeValueMemberS{Value: transactionID},
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
func (r *DynamoTransactionRepository) UpdateTransaction(ctx context.Context, accountID, transactionID string, values *models.Transaction) (*dynamodb.UpdateItemOutput, error) {
	// Validate input using config keys
	if accountID == "" {
		return nil, fmt.Errorf("%s cannot be empty", config.DBConfig.Keys.PartitionKey)
	}
	if transactionID == "" {
		return nil, fmt.Errorf("%s cannot be empty", config.DBConfig.Keys.SortKey)
	}

	// Convert struct to a map for partial updates
	updates, err := values.TransactionUpdatePayload()
	if err != nil {
		return nil, fmt.Errorf("failed to convert transaction to update map: %w", err)
	}
	fmt.Println("Transaction update payload: ", updates)

	// Ensure at least one field is provided for update
	if len(updates) == 0 {
		return nil, errors.New("no fields provided for update")
	}

	// Define key using config
	key := map[string]types.AttributeValue{
		config.DBConfig.Keys.PartitionKey: &types.AttributeValueMemberS{Value: accountID},
		config.DBConfig.Keys.SortKey:      &types.AttributeValueMemberS{Value: transactionID},
	}

	// Call DynamoDB update function with key
	result, err := r.DB.UpdateItem(ctx, key, updates)
	if err != nil {
		fmt.Printf("Error updating transaction, %s", err)
		return nil, err
	}

	fmt.Printf("Transaction updated: %s | UpdatedFields: %v\n", transactionID, result.Attributes)
	return result, nil
}

// DeleteTransaction removes a transaction using configured keys
func (r *DynamoTransactionRepository) DeleteTransaction(ctx context.Context, accountID, transactionID string) error {
	// Validate input using config keys
	if accountID == "" {
		return fmt.Errorf("%s cannot be empty", config.DBConfig.Keys.PartitionKey)
	}
	if transactionID == "" {
		return fmt.Errorf("%s cannot be empty", config.DBConfig.Keys.SortKey)
	}

	// Define key using config
	key := map[string]types.AttributeValue{
		config.DBConfig.Keys.PartitionKey: &types.AttributeValueMemberS{Value: accountID},
		config.DBConfig.Keys.SortKey:      &types.AttributeValueMemberS{Value: transactionID},
	}

	// Call DynamoDB delete function
	_, err := r.DB.DeleteItem(ctx, key)
	if err != nil {
		return err
	}

	fmt.Printf("Transaction deleted: %s\n", transactionID)
	return nil
}

func (r *DynamoTransactionRepository) UpdateFraudTransaction(ctx context.Context, phoneNumber string, isFraud bool) error {
	potentialFrauds, err := r.GetFraudTransaction(ctx, phoneNumber)
	if err != nil {
		return err
	}
	if len(potentialFrauds) == 0 {
		return errors.New("No Potential frauds for number: " + phoneNumber)
	}

	var wg sync.WaitGroup
	errorResults := make(chan error, len(potentialFrauds))
	for _, msg := range potentialFrauds {
		wg.Add(1)
		go func(txn models.Transaction) {

			defer wg.Done()

			if isFraud {
				txn.TransactionStatus = "FRAUD"
				_, err := r.UpdateTransaction(
					ctx,
					txn.AccountID,
					txn.TransactionID,
					&txn,
				)
				if err != nil {
					fmt.Printf("Error updating transaction with phone number: %s to fraud. Error: %s", phoneNumber, err)
					errorResults <- err
				}

			} else {
				txn.TransactionStatus = "APPROVED"
				_, err := r.UpdateTransaction(
					ctx,
					txn.AccountID,
					txn.TransactionID,
					&txn,
				)
				if err != nil {
					fmt.Printf("Error updating transaction with phone number: %s to Approved. Error: %s", phoneNumber, err)
					errorResults <- err
				}
			}
		}(msg)
	}
	wg.Wait()
	close(errorResults)
	return middleware.MergeErrors(errorResults)
}
