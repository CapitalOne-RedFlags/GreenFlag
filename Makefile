fraud_pipeline:
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./cmd/lambda/bootstrap ./cmd/lambda/fraud_pipeline.go
	zip ./cmd/lambda/fraud_pipeline.zip ./cmd/lambda/bootstrap
	rm -f ./cmd/lambda/bootstrap