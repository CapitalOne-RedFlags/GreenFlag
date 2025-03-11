# Variables
GOOS=linux
GOARCH=amd64

# Ensure we always resolve the project root correctly
ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))../)

BUILD_DIR=$(ROOT_DIR)/deployments/build
TRANSACTION_BUILD_DIR=$(BUILD_DIR)/transaction_pipeline
FRAUD_BUILD_DIR=$(BUILD_DIR)/fraud_pipeline

TRANSACTION_SRC=$(ROOT_DIR)/cmd/lambda/transactions
FRAUD_SRC=$(ROOT_DIR)/cmd/lambda/fraud

TRANSACTION_BINARY=$(TRANSACTION_BUILD_DIR)/bootstrap
FRAUD_BINARY=$(FRAUD_BUILD_DIR)/bootstrap

TRANSACTION_ZIP=$(TRANSACTION_BUILD_DIR).zip
FRAUD_ZIP=$(FRAUD_BUILD_DIR).zip

TRANSACTION_FUNCTION=TransactionPipelineFunction
FRAUD_FUNCTION=FraudPipelineFunction
STACK_NAME=TransactionConsumerStack
PROFILE=AdministratorAccess-140023383737
TEMPLATE_FILE=$(ROOT_DIR)/deployments/template.yaml

# Ensure build directories exist
.PHONY: build-dir
build-dir:
	mkdir -p $(TRANSACTION_BUILD_DIR)
	mkdir -p $(FRAUD_BUILD_DIR)

# Debugging target to check variable values
.PHONY: debug
debug:
	@echo "ROOT_DIR: $(ROOT_DIR)"
	@echo "TRANSACTION_BUILD_DIR: $(TRANSACTION_BUILD_DIR)"
	@echo "FRAUD_BUILD_DIR: $(FRAUD_BUILD_DIR)"
	@echo "TRANSACTION_BINARY: $(TRANSACTION_BINARY)"
	@echo "FRAUD_BINARY: $(FRAUD_BINARY)"

# Build TransactionPipelineFunction
.PHONY: build-transaction
build-transaction: build-dir
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(TRANSACTION_BINARY) $(TRANSACTION_SRC)/transaction_pipeline.go

# Build FraudPipelineFunction
.PHONY: build-fraud
build-fraud: build-dir
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -tags lambda.norpc -o $(FRAUD_BINARY) $(FRAUD_SRC)/fraud_pipeline.go

# Build all functions
.PHONY: build
build: build-transaction build-fraud

# Package TransactionPipelineFunction
.PHONY: package-transaction
package-transaction: build-transaction
	cd $(TRANSACTION_BUILD_DIR) && zip -r $(TRANSACTION_ZIP) bootstrap

# Package FraudPipelineFunction
.PHONY: package-fraud
package-fraud: build-fraud
	cd $(FRAUD_BUILD_DIR) && zip -r $(FRAUD_ZIP) bootstrap

# Deploy TransactionPipelineFunction using AWS CLI
.PHONY: deploy-transaction
deploy-transaction: package-transaction
	aws lambda update-function-code --function-name $(TRANSACTION_FUNCTION) --zip-file fileb://$(TRANSACTION_ZIP)

# Deploy FraudPipelineFunction using AWS CLI
.PHONY: deploy-fraud
deploy-fraud: package-fraud
	aws lambda update-function-code --function-name $(FRAUD_FUNCTION) --zip-file fileb://$(FRAUD_ZIP)

# Deploy both functions using AWS CLI
.PHONY: deploy
deploy: deploy-transaction deploy-fraud

# Deploy using AWS SAM after building and packaging
.PHONY: deploy-sam
deploy-sam: build package
	sam sync --stack-name $(STACK_NAME) --watch -t $(TEMPLATE_FILE) --profile $(PROFILE)

# Package all functions before deployment
.PHONY: package
package: package-transaction package-fraud

# Clean up: Delete CloudFormation stack and remove binaries
.PHONY: clean
clean:
	aws cloudformation delete-stack --stack-name $(STACK_NAME) --profile $(PROFILE)
	rm -rf $(BUILD_DIR)
