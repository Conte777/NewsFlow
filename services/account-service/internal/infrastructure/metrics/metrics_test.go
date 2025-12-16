package metrics

import (
	"testing"
)

// TestMetrics_RecordSubscription tests subscription recording
func TestMetrics_RecordSubscription(t *testing.T) {
	// Use the global DefaultMetrics instance
	// Record a subscription
	DefaultMetrics.RecordSubscription(1.5)

	// This test verifies that the method doesn't panic
	// Actual metric values are tested via Prometheus scraping in integration tests
}

// TestMetrics_RecordSubscriptionError tests subscription error recording
func TestMetrics_RecordSubscriptionError(t *testing.T) {
	// Record errors with different error types
	DefaultMetrics.RecordSubscriptionError("repository_error")
	DefaultMetrics.RecordSubscriptionError("no_active_accounts")
	DefaultMetrics.RecordSubscriptionError("") // Test empty error type

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordNewsCollection tests news collection recording
func TestMetrics_RecordNewsCollection(t *testing.T) {
	// Record news collection with positive items
	DefaultMetrics.RecordNewsCollection(5, 2.3)

	// Test with zero items (should not panic)
	DefaultMetrics.RecordNewsCollection(0, 1.0)

	// Test with negative items (should not panic, value won't be added)
	DefaultMetrics.RecordNewsCollection(-1, 1.5)

	// This test verifies that the method doesn't panic
}

// TestMetrics_UpdateAccounts tests account metrics update
func TestMetrics_UpdateAccounts(t *testing.T) {
	// Update account metrics
	DefaultMetrics.UpdateAccounts(3, 5)

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordKafkaMessage tests Kafka message recording
func TestMetrics_RecordKafkaMessage(t *testing.T) {
	// Record Kafka messages with duration
	DefaultMetrics.RecordKafkaMessage(0.001)
	DefaultMetrics.RecordKafkaMessage(0.05)
	DefaultMetrics.RecordKafkaMessage(0.1)

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordKafkaError tests Kafka error recording
func TestMetrics_RecordKafkaError(t *testing.T) {
	// Record Kafka errors with different error types
	DefaultMetrics.RecordKafkaError("send_failed")
	DefaultMetrics.RecordKafkaError("timeout")
	DefaultMetrics.RecordKafkaError("") // Test empty error type

	// This test verifies that the method doesn't panic
}

// TestMetrics_UpdateActiveSubscriptions tests subscription update
func TestMetrics_UpdateActiveSubscriptions(t *testing.T) {
	// Update subscriptions
	DefaultMetrics.UpdateActiveSubscriptions(10)

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordUnsubscription tests unsubscription recording
func TestMetrics_RecordUnsubscription(t *testing.T) {
	// Record unsubscription
	DefaultMetrics.RecordUnsubscription(0.5)

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordUnsubscriptionError tests unsubscription error recording
func TestMetrics_RecordUnsubscriptionError(t *testing.T) {
	// Record errors with different error types
	DefaultMetrics.RecordUnsubscriptionError("repository_error")
	DefaultMetrics.RecordUnsubscriptionError("leave_failed")
	DefaultMetrics.RecordUnsubscriptionError("")

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordAccountReconnection tests account reconnection recording
func TestMetrics_RecordAccountReconnection(t *testing.T) {
	// Record reconnection events
	DefaultMetrics.RecordAccountReconnection()
	DefaultMetrics.RecordAccountReconnection()

	// This test verifies that the method doesn't panic
}

// TestMetrics_RecordAccountRateLimit tests account rate limit recording
func TestMetrics_RecordAccountRateLimit(t *testing.T) {
	// Record rate limit events
	DefaultMetrics.RecordAccountRateLimit()

	// This test verifies that the method doesn't panic
}

// TestDefaultMetrics_Initialized verifies DefaultMetrics initialization
func TestDefaultMetrics_Initialized(t *testing.T) {
	// Verify DefaultMetrics is initialized
	if DefaultMetrics == nil {
		t.Fatal("DefaultMetrics should be initialized")
	}

	// Verify subscription metrics are non-nil
	if DefaultMetrics.SubscriptionsTotal == nil {
		t.Error("SubscriptionsTotal should not be nil")
	}
	if DefaultMetrics.SubscriptionErrors == nil {
		t.Error("SubscriptionErrors should not be nil")
	}
	if DefaultMetrics.UnsubscriptionErrors == nil {
		t.Error("UnsubscriptionErrors should not be nil")
	}

	// Verify account metrics are non-nil
	if DefaultMetrics.ActiveAccounts == nil {
		t.Error("ActiveAccounts should not be nil")
	}
	if DefaultMetrics.AccountReconnections == nil {
		t.Error("AccountReconnections should not be nil")
	}
	if DefaultMetrics.AccountRateLimits == nil {
		t.Error("AccountRateLimits should not be nil")
	}

	// Verify news collection metrics are non-nil
	if DefaultMetrics.NewsCollectionTotal == nil {
		t.Error("NewsCollectionTotal should not be nil")
	}
	if DefaultMetrics.NewsItemsCollected == nil {
		t.Error("NewsItemsCollected should not be nil")
	}

	// Verify Kafka metrics are non-nil
	if DefaultMetrics.KafkaMessagesProduced == nil {
		t.Error("KafkaMessagesProduced should not be nil")
	}
	if DefaultMetrics.KafkaProduceErrors == nil {
		t.Error("KafkaProduceErrors should not be nil")
	}
	if DefaultMetrics.KafkaProduceDuration == nil {
		t.Error("KafkaProduceDuration should not be nil")
	}
}
