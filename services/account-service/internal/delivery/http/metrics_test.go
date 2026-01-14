package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	// Import metrics package to ensure init() is called
	_ "github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/metrics"
)

// TestMetricsEndpoint verifies the /metrics endpoint is correctly configured
func TestMetricsEndpoint(t *testing.T) {
	// Create a test HTTP server with /metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	mux.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify content type
	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to contain 'text/plain', got %s", contentType)
	}

	// Verify response contains Prometheus metrics
	body := rec.Body.String()

	// Check for standard Go metrics (always present)
	if !strings.Contains(body, "go_goroutines") {
		t.Error("Expected response to contain 'go_goroutines' metric")
	}

	// Check for our custom metrics
	// Note: CounterVec metrics (with labels) only appear after first use with labels
	// so we only check for simple Counter and Gauge metrics here
	expectedMetrics := []string{
		"account_service_subscriptions_total",
		"account_service_unsubscriptions_total",
		"account_service_active_subscriptions",
		"account_service_news_collections_total",
		"account_service_news_items_sent_total",
		"account_service_active_accounts",
		"account_service_total_accounts",
		"account_service_account_reconnections_total",
		"account_service_account_rate_limits_total",
		"account_service_kafka_messages_produced_total",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Expected response to contain metric '%s'", metric)
		}
	}
}

// TestMetricsEndpointAcceptsGET verifies GET requests work
func TestMetricsEndpointAcceptsGET(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Should return 200 OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d for GET request, got %d", http.StatusOK, rec.Code)
	}
}
