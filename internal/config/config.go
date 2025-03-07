package config

import "os"

type Config struct {
    DBConfig     DBConfig
    SQSConfig    SQSConfig
}

type SQSConfig struct {
    QueueURL string
}

type DBConfig struct {
    AllowedUpdateFields map[string]bool
}

var AppConfig Config

func init() {
    // Initialize SQS configuration
    AppConfig.SQSConfig = SQSConfig{
        QueueURL: "https://sqs.us-east-1.amazonaws.com/YOUR_ACCOUNT_ID/bank-transactions",
    }

    // Initialize DB configuration
    AppConfig.DBConfig = DBConfig{
        AllowedUpdateFields: map[string]bool{
            "TransactionStatus": true,
            // Add other allowed fields here
        },
    }
}
