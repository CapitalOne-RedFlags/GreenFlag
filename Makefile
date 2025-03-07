.PHONY: publish-transactions
publish-transactions:
	go run cmd/csv_publisher/main.go

.PHONY: build-lambda
build-lambda:
	GOOS=linux GOARCH=amd64 go build -o bin/lambda cmd/lambda/main.go

.PHONY: deploy-lambda
deploy-lambda: build-lambda
	sam deploy --template-file deployments/template.yaml
