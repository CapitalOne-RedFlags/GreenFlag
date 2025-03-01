# GreenFlag Architecture Design

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
│   ├── models/
│   │   ├── transaction.go  # Transaction model
│   │   ├── account.go  # Account model
│   ├── services/
│   │   ├── transaction_service.go  # Transaction processing
│   │   ├── fraud_service.go  # Fraud detection logic
│   │   ├── notification_service.go  # Handles SNS alerts
│   ├── events/
│   │   ├── event_types.go  # Defines different event types
│   │   ├── event_dispatcher.go  # Central logic for publishing events, uses event_bus
|   |   ├── event_bus.go  # Utilized by event_dispatcher to invoke appropriate methods from internal/messaging
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

## Core Dependency Flow
```bash
handlers → services → events → messaging → db
```
| **Component**            | **Depends On**                   | **Purpose** | **Example** |
|-------------------------|--------------------------------|-------------|------------|
| `cmd/lambda/main.go`     | `handlers/`                     | Entry point for AWS Lambda. | The `main.go` file starts the Lambda function and calls `TransactionHandler` when an event is received. |
| `handlers/`             | `services/`, `events/`          | Processes Lambda requests and routes them. | The `TransactionHandler` in `internal/handlers/transaction_handler.go` should receive an event from AWS Lambda (triggered by SQS or API Gateway), parse it, and pass it to `event_handlers.go` for processing. |
| `services/`             | `events/`, `db/`, `messaging/`  | Business logic layer (transaction processing, fraud detection). | `ProcessTransaction` in `transaction_service.go` checks for fraud, stores the transaction in DynamoDB, and dispatches a `TransactionCreated` event. |
| `events/`               | `messaging/`                    | Defines events, publishes them, and processes incoming events. | The `event_handlers.go` file routes a `TransactionCreated` event to `ProcessTransaction`, and `event_dispatcher.go` sends it to SNS or SQS via the `event_bus`. |
| `messaging/`            | AWS SDK (`sns.go`, `sqs.go`)    | Handles actual AWS messaging logic. | `sns.go` publishes fraud alerts to an SNS topic, while `sqs.go` receives messages from the transaction queue. |
| `db/`                   | AWS SDK (`dynamo.go`)           | Manages data persistence in DynamoDB. | `dynamo.go` saves new transactions in the `Transactions` table and retrieves them when needed. |

## Program Flow
![Green Flag Flow](docs/GreenFlag_Flow.png)

## High Level Diagram
![Green Flag Services](docs/)
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



