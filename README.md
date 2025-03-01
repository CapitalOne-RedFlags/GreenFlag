# Project Name: Event-Driven Serverless Architecture

## Overview
This repository contains a **serverless event-driven architecture** built using **AWS Lambda, Amazon SQS, Amazon SNS, and DynamoDB**. The system is designed to process transactions, detect fraud, and notify users via email/SMS.

### **Key Features**
- **AWS Lambda:** Runs stateless functions triggered by SQS messages, SNS notifications, or API Gateway requests.
- **Amazon SQS:** Used for queueing transactions before processing.
- **Amazon SNS:** Sends notifications to users regarding fraud alerts.
- **Amazon DynamoDB:** Stores transactions and account details efficiently.
- **Event-Driven Processing:** Uses an `events` package to manage different types of events.
- **Structured Logging:** Uses a `logging` package for observability.
- **Infrastructure as Code:** Deployment is handled using AWS SAM (Serverless Application Model).

## **Project Structure**
```
project-root/
├── cmd/
│   └── lambda/
│       └── main.go  # Entry point for Lambda function
├── internal/
│   ├── config/
│   │   └── config.go  # Handles app config & environment variables
│   ├── db/
│   │   ├── repository.go  # Generic DB repository interface
│   │   ├── dynamo.go  # DynamoDB implementation
│   ├── messaging/
│   │   ├── sns.go  # SNS message publisher
│   │   ├── sqs.go  # SQS consumer for event messages
│   │   ├── event_bus.go  # Event handling abstraction
│   ├── models/
│   │   ├── transaction.go  # Transaction model
│   │   ├── account.go  # Account model
│   ├── services/
│   │   ├── transaction_service.go  # Transaction processing
│   │   ├── fraud_service.go  # Fraud detection logic
│   │   ├── notification_service.go  # Handles SNS alerts
│   ├── events/
│   │   ├── event_types.go  # Defines different event types
│   │   ├── event_dispatcher.go  # Central logic for publishing events
│   │   ├── event_handlers.go  # Handlers for processing incoming events
│   ├── handlers/
│   │   ├── transaction_handler.go  # Handles Lambda triggers for transactions
│   │   ├── fraud_handler.go  # Processes fraud-related events
│   │   ├── response_handler.go  # Handles user Yes/No responses from SNS
│   ├── logging/
│   │   └── logger.go  # Implements structured logging
│   ├── middleware/
│   │   ├── error_handler.go  # Centralized error handling
│   ├── observability/
│   │   ├── tracing.go  # OpenTelemetry (optional)
│   │   ├── metrics.go  # AWS CloudWatch metrics collection
├── test/
│   ├── events_test.go  # Test event-driven logic
│   ├── db_test.go  # Test DynamoDB logic
│   ├── fraud_test.go  # Test fraud detection
│   ├── transaction_test.go  # Test transactions
│   ├── messaging_test.go  # Test SNS & SQS messaging
├── deployments/
│   ├── template.yaml  # AWS SAM configuration file
│   ├── samconfig.toml  # SAM deployment config
├── go.mod
├── go.sum
├── Makefile  # Build, test, deploy commands
```

## **Getting Started**
### **Prerequisites**
Ensure you have the following installed:
- **Go 1.x** (for building the Lambda function)
- **AWS CLI** (for interacting with AWS services)
- **AWS SAM CLI** (for deploying the infrastructure)
- **Docker** (optional, for testing Lambda locally)

### **Installation & Setup**
1. Clone the repository
2. Initialize Go modules:
   ```sh
   go mod tidy
   ```

## **Deployment**
The project uses AWS **SAM (Serverless Application Model)** for deployment.

### **1️⃣ Build the Application**
```sh
sam build --template-file deployments/template.yaml
```

### **2️⃣ Deploy to AWS**
```sh
sam deploy --template-file deployments/template.yaml --guided
```
This will:
- Package and upload the Lambda function.
- Create an SQS queue, SNS topic, and DynamoDB table.
- Deploy necessary IAM roles and permissions.


### **3️⃣ Testing Locally**
#### **Test Lambda Function Locally**
```sh
sam local invoke TransactionProcessor --event test_event.json
```

#### **Manually Send a Transaction to SQS**
```sh
aws sqs send-message --queue-url <YOUR_QUEUE_URL> --message-body '{"transactionID": "123", "amount": 50}'
```

#### **Query DynamoDB Table**
```sh
aws dynamodb scan --table-name Transactions
```

## **Monitoring & Logs**
To check logs for AWS Lambda:
```sh
aws logs tail /aws/lambda/TransactionProcessor --follow
```



