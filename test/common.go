package test

import (
	"context"
	"testing"
	"time"

	"github.com/CapitalOne-RedFlags/GreenFlag/internal/models"
	"github.com/google/uuid"
)

func GetTestTransaction(email string) models.Transaction {
	return models.Transaction{
		TransactionID:           uuid.New().String(),
		AccountID:               "TEST-" + uuid.New().String(),
		TransactionAmount:       100.50,
		TransactionDate:         time.Now().Format(time.RFC3339),
		TransactionType:         "PURCHASE",
		Location:                "New York",
		DeviceID:                "device-123",
		IPAddress:               "192.168.1.1",
		MerchantID:              "merchant-456",
		Channel:                 "WEB",
		CustomerAge:             30,
		CustomerOccupation:      "Engineer",
		TransactionDuration:     120,
		LoginAttempts:           1,
		AccountBalance:          5000.00,
		PreviousTransactionDate: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		PhoneNumber:             "+12025550179",
		Email:                   email,
		TransactionStatus:       "PENDING",
	}
}

// Wrapper function for a test to fail after specified amount of time
func RunTestWithCustomTimeout(t *testing.T, seconds time.Duration, name string, testFunc func(*testing.T)) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), seconds*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)

		t.Run(name, testFunc)
	}()
	select {
	case <-done:

	case <-ctx.Done():
		t.Fatal("Test timed out")
	}
}
