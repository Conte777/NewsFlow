package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// HealthChecker defines interface for components that can report their health
type HealthChecker interface {
	// HealthCheck returns true if component is healthy, false otherwise
	HealthCheck(ctx context.Context) bool
}

// AccountManagerHealthChecker wraps account manager to check if any accounts are available
type AccountManagerHealthChecker interface {
	GetActiveAccountCount() int
}

// KafkaHealthChecker wraps Kafka components to check connectivity
type KafkaHealthChecker interface {
	IsHealthy() bool
}

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents health status of a single component
type ComponentHealth struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the JSON response for health check
type HealthResponse struct {
	Status     HealthStatus      `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Components []ComponentHealth `json:"components"`
}

// HealthHandler handles HTTP health check requests
type HealthHandler struct {
	accountManager AccountManagerHealthChecker
	kafkaProducer  KafkaHealthChecker
	kafkaConsumer  KafkaHealthChecker
	logger         zerolog.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(
	accountManager AccountManagerHealthChecker,
	kafkaProducer KafkaHealthChecker,
	kafkaConsumer KafkaHealthChecker,
	logger zerolog.Logger,
) *HealthHandler {
	return &HealthHandler{
		accountManager: accountManager,
		kafkaProducer:  kafkaProducer,
		kafkaConsumer:  kafkaConsumer,
		logger:         logger,
	}
}

// ServeHTTP implements http.Handler interface
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Add timeout for health check to prevent hanging
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check health of all components
	components := h.checkComponents(ctx)

	// Determine overall status
	status := h.determineOverallStatus(components)

	response := HealthResponse{
		Status:     status,
		Timestamp:  time.Now().UTC(),
		Components: components,
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if status == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if status == HealthStatusDegraded {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	// Log health status
	logEvent := h.logger.Debug()
	if status == HealthStatusUnhealthy {
		logEvent = h.logger.Warn()
	} else if status == HealthStatusDegraded {
		logEvent = h.logger.Info()
	}
	logEvent.
		Str("status", string(status)).
		Int("status_code", statusCode).
		Interface("components", components).
		Msg("Health check completed")

	// Write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Encode response
	// Note: Cannot call http.Error after WriteHeader, only log errors
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode health check response")
		// Headers already sent, cannot change response
		return
	}
}

// checkComponents checks health of all service components
// Context can be used for timeout or cancellation of health checks
func (h *HealthHandler) checkComponents(ctx context.Context) []ComponentHealth {
	components := make([]ComponentHealth, 0, 3)

	// Check context before starting checks
	select {
	case <-ctx.Done():
		// Context cancelled or timed out
		return []ComponentHealth{{
			Name:    "health_check",
			Healthy: false,
			Message: "Health check timeout",
		}}
	default:
	}

	// Check Account Manager
	accountCount := h.accountManager.GetActiveAccountCount()
	accountHealthy := accountCount > 0
	accountMsg := ""
	if !accountHealthy {
		accountMsg = "No active Telegram accounts available"
	}

	components = append(components, ComponentHealth{
		Name:    "telegram_accounts",
		Healthy: accountHealthy,
		Message: accountMsg,
	})

	// Check Kafka Producer
	producerHealthy := h.kafkaProducer.IsHealthy()
	producerMsg := ""
	if !producerHealthy {
		producerMsg = "Kafka producer is not healthy"
	}

	components = append(components, ComponentHealth{
		Name:    "kafka_producer",
		Healthy: producerHealthy,
		Message: producerMsg,
	})

	// Check Kafka Consumer
	consumerHealthy := h.kafkaConsumer.IsHealthy()
	consumerMsg := ""
	if !consumerHealthy {
		consumerMsg = "Kafka consumer is not healthy"
	}

	components = append(components, ComponentHealth{
		Name:    "kafka_consumer",
		Healthy: consumerHealthy,
		Message: consumerMsg,
	})

	return components
}

// determineOverallStatus determines overall health status based on component health
func (h *HealthHandler) determineOverallStatus(components []ComponentHealth) HealthStatus {
	allHealthy := true
	anyHealthy := false

	for _, component := range components {
		if !component.Healthy {
			allHealthy = false
		} else {
			anyHealthy = true
		}
	}

	if allHealthy {
		return HealthStatusHealthy
	} else if anyHealthy {
		return HealthStatusDegraded
	}

	return HealthStatusUnhealthy
}
