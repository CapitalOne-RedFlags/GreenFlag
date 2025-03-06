package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBClient wraps the AWS SDK client
type DynamoDBClient struct {
	Client    *dynamodb.Client
	TableName string
}

func NewDynamoDBClient(client *dynamodb.Client, tableName string) *DynamoDBClient {
	return &DynamoDBClient{
		Client:    client,
		TableName: tableName,
	}
}

// PutItem inserts an item into DynamoDB and returns metadata.
func (d *DynamoDBClient) PutItem(ctx context.Context, item map[string]types.AttributeValue) (*dynamodb.PutItemOutput, string, error) {
	output, err := d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(d.TableName),
		ConditionExpression: aws.String(fmt.Sprintf(
			"attribute_not_exists(%s) AND attribute_not_exists(%s)",
			config.DBConfig.Keys.PartitionKey,
			config.DBConfig.Keys.SortKey,
		)),
		ReturnConsumedCapacity:      types.ReturnConsumedCapacityTotal,
		ReturnItemCollectionMetrics: types.ReturnItemCollectionMetricsSize,
	})

	// Handle duplicate transaction error
	var conditionCheckErr *types.ConditionalCheckFailedException
	if err != nil {
		if errors.As(err, &conditionCheckErr) {
			return nil, "", fmt.Errorf("transaction already exists")
		}
		return nil, "", fmt.Errorf("failed to put item: %w", err)
	}

	// Extract metadata as a JSON string for logging
	metadata, err := json.MarshalIndent(output.ConsumedCapacity, "", "  ")
	if err != nil {
		return output, "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	return output, string(metadata), nil
}

// GetItem retrieves an item from DynamoDB by primary key
func (d *DynamoDBClient) GetItem(ctx context.Context, key map[string]types.AttributeValue) (map[string]types.AttributeValue, error) {
	result, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &d.TableName,
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from DynamoDB: %w", err)
	}
	if result.Item == nil {
		return nil, fmt.Errorf("item not found")
	}
	return result.Item, nil
}

// UpdateItem updates a transaction in DynamoDB, safely constructs NoSQL query to update. **MAKE SURE YOUR UPDATES WERE VALIDATED WITH TRANSACTION'S PAYLOAD FUNCTION!**
func (d *DynamoDBClient) UpdateItem(ctx context.Context, key map[string]types.AttributeValue, updates map[string]interface{}) (*dynamodb.UpdateItemOutput, error) {

	var updateExprBuilder strings.Builder
	updateExprBuilder.WriteString("SET ")

	parts := []string{}
	exprAttrValues := make(map[string]types.AttributeValue)
	exprAttrNames := make(map[string]string)
	i := 0

	for field, value := range updates {
		// Use safe placeholders for attribute names & values
		placeholder := fmt.Sprintf("#F%d", i)
		valuePlaceholder := fmt.Sprintf(":V%d", i)

		parts = append(parts, fmt.Sprintf("%s = %s", placeholder, valuePlaceholder))
		exprAttrNames[placeholder] = field

		// Marshal the value to a DynamoDB attribute value
		attrValue, err := attributevalue.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value for field '%s': %w", field, err)
		}
		exprAttrValues[valuePlaceholder] = attrValue
		i++
	}

	// Join the update parts and complete the update expression
	updateExprBuilder.WriteString(strings.Join(parts, ", "))
	updateExpr := updateExprBuilder.String()

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(d.TableName),
		Key:                       key,
		UpdateExpression:          aws.String(updateExpr),
		ConditionExpression:       aws.String(config.DBConfig.UpdateCondition),
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ReturnValues:              types.ReturnValueUpdatedNew,
	}

	result, err := d.Client.UpdateItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	return result, nil
}

func (d *DynamoDBClient) DeleteItem(ctx context.Context, key map[string]types.AttributeValue) (*dynamodb.DeleteItemOutput, error) {
	result, err := d.Client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &d.TableName,
		Key:       key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete item from DynamoDB: %w", err)
	}
	return result, nil
}
