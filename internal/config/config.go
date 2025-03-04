package config

// DynamoDBConfig stores table settings
type DynamoDBConfig struct {
	TableName        string
	AllowedUpdateFields    map[string]bool
	UpdateCondition  string
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
}
