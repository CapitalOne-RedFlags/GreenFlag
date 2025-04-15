# Variables
GOOS=linux
GOARCH=amd64

# Resolve the project root (assumes the Makefile is in the project root)
ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

BUILD_DIR=$(ROOT_DIR)/deployments/build
TRANSACTION_BUILD_DIR=$(BUILD_DIR)/transaction_pipeline
FRAUD_BUILD_DIR=$(BUILD_DIR)/fraud_pipeline

TRANSACTION_SRC=$(ROOT_DIR)/cmd/lambda/transactions
FRAUD_SRC=$(ROOT_DIR)/cmd/lambda/fraud

TRANSACTION_FUNCTION=TransactionPipelineFunction
FRAUD_FUNCTION=FraudPipelineFunction
STACK_NAME=TransactionConsumerStack
PROFILE=CS620_C1_Capstone_Rex
TEMPLATE_FILE=$(ROOT_DIR)/deployments/template.yaml


# Debug target to display variable values
.PHONY: debug
debug:
	@echo "ROOT_DIR: $(ROOT_DIR)"
	@echo "TRANSACTION_BUILD_DIR: $(TRANSACTION_BUILD_DIR)"
	@echo "FRAUD_BUILD_DIR: $(FRAUD_BUILD_DIR)"

# Build TransactionPipelineFunction binary
.PHONY: build-TransactionPipelineFunction
build-TransactionPipelineFunction:
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda/transactions/transaction_pipeline.go

# Build FraudPipelineFunction binary
.PHONY: build-FraudPipelineFunction
build-FraudPipelineFunction:
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda/fraud/fraud_pipeline.go

# Build TransactionPipelineRetryFunction binary
.PHONY: build-TransactionPipelineRetryFunction
build-TransactionPipelineRetrFunction:
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda/transactions/transaction_pipeline.go

# Build FraudPipelineRetryFunction binary
.PHONY: build-FraudPipelineRetryFunction
build-FraudPipelineRetryFunction:
	mkdir -p $(ARTIFACTS_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap ./cmd/lambda/retry/fraud_retry_pipeline.go

# Build both functions (invoked by SAM during 'sam build')
.PHONY: build
build: build-transaction build-fraud build-TransactionPipelineRetryFunction build-FraudPipelineRetryFunction

# Run sam build to trigger the Makefile integration.
.PHONY: sam-build
sam-build:
	sam build -t $(TEMPLATE_FILE) --profile $(PROFILE)

# Deploy the stack using SAM deploy
.PHONY: deploy
deploy: sam-build
	sam deploy --stack-name $(STACK_NAME) --capabilities CAPABILITY_IAM --profile $(PROFILE) --no-confirm-changeset

# Use sam sync for iterative deployments during development
.PHONY: sync
sync:
	sam sync --stack-name $(STACK_NAME) --watch -t $(TEMPLATE_FILE) --profile $(PROFILE)

# Clean up: Delete the CloudFormation stack and remove build artifacts
.PHONY: clean
clean:
	aws cloudformation delete-stack --stack-name $(STACK_NAME) --profile $(PROFILE)
	rm -rf $(BUILD_DIR)
