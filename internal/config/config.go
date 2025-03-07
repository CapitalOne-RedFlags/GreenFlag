package config

// DynamoDBConfig stores table settings
type DynamoDBConfig struct {
	TableName           string
	AllowedUpdateFields map[string]bool
	UpdateCondition     string
	// Add key configuration
	Keys struct {
		PartitionKey string
		SortKey      string
	}
}

var DBConfig = &DynamoDBConfig{
	TableName: "TransactionsTable",
	AllowedUpdateFields: map[string]bool{
		"TransactionStatus":       true,
		"TransactionAmount":       true,
		"TransactionDate":         true,
		"Location":                true,
		"DeviceID":                true,
		"IPAddress":               true,
		"MerchantID":              true,
		"Channel":                 true,
		"CustomerAge":             true,
		"CustomerOccupation":      true,
		"TransactionDuration":     true,
		"LoginAttempts":           true,
		"AccountBalance":          true,
		"PreviousTransactionDate": true,
		"PhoneNumber":             true,
		"Email":                   true,
	},
	UpdateCondition: "TransactionStatus = :pending", // Only update if Pending
	Keys: struct {
		PartitionKey string
		SortKey      string
	}{
		PartitionKey: "AccountID",
		SortKey:      "TransactionID",
	}
}

func IsCI() bool {
	return GetEnv("CI", "false") == "true"
}

func PrintDBConfig() {
	fmt.Printf("DynamoDB Table: %s\n", DBConfig.TableName)
	fmt.Printf("DynamoDB Endpoint: %s\n", DBConfig.DynamoDBEndpoint)
	fmt.Printf("AWS Region: %s\n", GetEnv("AWS_REGION", "us-east-1"))
	fmt.Printf("CI Mode: %v\n", IsCI())
}
