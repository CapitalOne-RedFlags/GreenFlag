package observability

import (
	"context"
	"fmt"

	"github.com/aws/aws-xray-sdk-go/xray"
)

// Metadata keys as constants for better maintainability
const (
	// Fraud-related metadata keys
	KeyFraudDetected            = "FraudDetected"
	KeyFraudCount               = "FraudCount"
	KeyFraudulentTransactionIDs = "FraudulentTransactionIDs"
	KeyFraudulentEmails         = "FraudulentEmails"
	KeyFraudulentAmounts        = "FraudulentAmounts"
	KeyEmailsChecked            = "EmailsChecked"

	// Transaction-related metadata keys
	KeyEventRecordsCount = "EventRecordsCount"
	KeyTransactionIDs    = "TransactionIDs"
	KeyAccountIDs        = "AccountIDs"
	KeyEmails            = "Emails"
	KeyTransaction       = "Transaction-"
)

// SafeAddMetadata adds metadata to an X-Ray segment with error handling
func SafeAddMetadata(seg *xray.Segment, key string, value interface{}) {
	if err := seg.AddMetadata(key, value); err != nil {
		fmt.Printf("Failed to add metadata [%s]: %v\n", key, err)
	}
}

// SafeAddError adds an error to an X-Ray segment with error handling
func SafeAddError(seg *xray.Segment, err error) {
	if addErr := seg.AddError(err); addErr != nil {
		fmt.Printf("Failed to add error to segment: %v\n", addErr)
	}
}

// SafeAddAnnotation adds an annotation to X-Ray context with error handling
func SafeAddAnnotation(ctx context.Context, key string, value string) {
	if err := xray.AddAnnotation(ctx, key, value); err != nil {
		fmt.Printf("Failed to add annotation [%s]: %v\n", key, err)
	}
}
