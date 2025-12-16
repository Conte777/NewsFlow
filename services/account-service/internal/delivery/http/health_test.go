package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// mockAccountManager implements AccountManagerHealthChecker for testing
type mockAccountManager struct {
	activeCount int
}

func (m *mockAccountManager) GetActiveAccountCount() int {
	return m.activeCount
}

// mockKafkaHealthChecker implements KafkaHealthChecker for testing
type mockKafkaHealthChecker struct {
	healthy bool
}

func (m *mockKafkaHealthChecker) IsHealthy() bool {
	return m.healthy
}

func TestHealthHandler_AllHealthy(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 2},
		&mockKafkaHealthChecker{healthy: true},
		&mockKafkaHealthChecker{healthy: true},
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != HealthStatusHealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusHealthy, response.Status)
	}

	if len(response.Components) != 3 {
		t.Errorf("Expected 3 components, got %d", len(response.Components))
	}

	// Check all components are healthy
	for _, comp := range response.Components {
		if !comp.Healthy {
			t.Errorf("Component %s should be healthy", comp.Name)
		}
	}
}

func TestHealthHandler_NoAccounts(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 0},
		&mockKafkaHealthChecker{healthy: true},
		&mockKafkaHealthChecker{healthy: true},
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert - should be degraded but not fail completely
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for degraded state, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != HealthStatusDegraded {
		t.Errorf("Expected status %s, got %s", HealthStatusDegraded, response.Status)
	}
}

func TestHealthHandler_AllUnhealthy(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 0},
		&mockKafkaHealthChecker{healthy: false},
		&mockKafkaHealthChecker{healthy: false},
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status %s, got %s", HealthStatusUnhealthy, response.Status)
	}

	// Check all components are unhealthy
	for _, comp := range response.Components {
		if comp.Healthy {
			t.Errorf("Component %s should be unhealthy", comp.Name)
		}
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 2},
		&mockKafkaHealthChecker{healthy: true},
		&mockKafkaHealthChecker{healthy: true},
		logger,
	)

	// Create POST request (should fail)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}

	// Check Allow header
	allowHeader := w.Header().Get("Allow")
	if allowHeader != http.MethodGet {
		t.Errorf("Expected Allow header %s, got %s", http.MethodGet, allowHeader)
	}
}

func TestHealthHandler_PartialDegradation_ProducerUnhealthy(t *testing.T) {
	// Setup - Producer unhealthy, consumer and accounts healthy
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 2},
		&mockKafkaHealthChecker{healthy: false}, // producer unhealthy
		&mockKafkaHealthChecker{healthy: true},  // consumer healthy
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert - should be degraded (not all healthy, but not all unhealthy)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for degraded, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != HealthStatusDegraded {
		t.Errorf("Expected status %s, got %s", HealthStatusDegraded, response.Status)
	}

	// Verify specific component states
	producerHealthy := false
	consumerHealthy := false
	for _, comp := range response.Components {
		if comp.Name == "kafka_producer" && !comp.Healthy {
			producerHealthy = false // Correct state
		}
		if comp.Name == "kafka_consumer" && comp.Healthy {
			consumerHealthy = true
		}
	}

	if producerHealthy {
		t.Error("Producer should be unhealthy")
	}
	if !consumerHealthy {
		t.Error("Consumer should be healthy")
	}
}

func TestHealthHandler_PartialDegradation_AccountsUnhealthy(t *testing.T) {
	// Setup - Accounts unhealthy, Kafka healthy
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 0},   // no accounts
		&mockKafkaHealthChecker{healthy: true}, // producer healthy
		&mockKafkaHealthChecker{healthy: true}, // consumer healthy
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert
	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != HealthStatusDegraded {
		t.Errorf("Expected status %s, got %s", HealthStatusDegraded, response.Status)
	}
}

func TestHealthHandler_ResponseHeaders(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 2},
		&mockKafkaHealthChecker{healthy: true},
		&mockKafkaHealthChecker{healthy: true},
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(w, req)

	// Assert Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestHealthHandler_TimestampIsRecent(t *testing.T) {
	// Setup
	logger := zerolog.Nop()
	handler := NewHealthHandler(
		&mockAccountManager{activeCount: 2},
		&mockKafkaHealthChecker{healthy: true},
		&mockKafkaHealthChecker{healthy: true},
		logger,
	)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Record time before request
	beforeTime := time.Now().UTC()

	// Execute
	handler.ServeHTTP(w, req)

	// Record time after request
	afterTime := time.Now().UTC()

	// Decode response
	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Assert timestamp is between beforeTime and afterTime
	if response.Timestamp.Before(beforeTime) || response.Timestamp.After(afterTime) {
		t.Errorf("Timestamp %v is not between %v and %v", response.Timestamp, beforeTime, afterTime)
	}
}
