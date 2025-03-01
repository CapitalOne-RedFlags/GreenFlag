#!/bin/bash

# Explanation in README
dirs=(
    "cmd/lambda"
    "internal/config"
    "internal/db"
    "internal/messaging"
    "internal/models"
    "internal/services"
    "internal/events"
    "internal/handlers"
    "internal/logging"
    "internal/middleware"
    "internal/observability"
    "test"
    "deployments"
)

files=(
    "cmd/lambda/main.go"
    "internal/config/config.go"
    "internal/db/repository.go"
    "internal/db/dynamo.go"
    "internal/messaging/sns.go"
    "internal/messaging/sqs.go"
    "internal/messaging/event_bus.go"
    "internal/models/transaction.go"
    "internal/models/account.go"
    "internal/services/transaction_service.go"
    "internal/services/fraud_service.go"
    "internal/services/notification_service.go"
    "internal/events/event_types.go"
    "internal/events/event_dispatcher.go"
    "internal/events/event_handlers.go"
    "internal/handlers/transaction_handler.go"
    "internal/handlers/fraud_handler.go"
    "internal/handlers/response_handler.go"
    "internal/logging/logger.go"
    "internal/middleware/error_handler.go"
    "internal/observability/tracing.go"
    "internal/observability/metrics.go"
    "test/events_test.go"
    "test/db_test.go"
    "test/fraud_test.go"
    "test/transaction_test.go"
    "test/messaging_test.go"
    "deployments/template.yaml"
    "deployments/samconfig.toml"
    "go.mod"
    "go.sum"
    "Makefile"
)

# Create directories
for dir in "${dirs[@]}"; do
    mkdir -p "$dir"
done

# Create files
for file in "${files[@]}"; do
    touch "$file"
done

echo "Project structure created successfully owob!"
