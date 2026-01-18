package http

import (
	"encoding/json"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"go.uber.org/fx"
)

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
	accountManager domain.AccountManager
	kafkaProducer  domain.KafkaProducer
	kafkaConsumer  domain.KafkaConsumer
	logger         zerolog.Logger
}

// HealthHandlerParams defines parameters for HealthHandler with optional dependencies
type HealthHandlerParams struct {
	fx.In

	AccountManager domain.AccountManager
	KafkaProducer  domain.KafkaProducer
	KafkaConsumer  domain.KafkaConsumer `optional:"true"`
	Logger         zerolog.Logger
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(params HealthHandlerParams) *HealthHandler {
	return &HealthHandler{
		accountManager: params.AccountManager,
		kafkaProducer:  params.KafkaProducer,
		kafkaConsumer:  params.KafkaConsumer,
		logger:         params.Logger,
	}
}

// Handle handles the health check request for fasthttp
func (h *HealthHandler) Handle(ctx *fasthttp.RequestCtx) {
	components := h.checkComponents()
	status := h.determineOverallStatus(components)

	response := HealthResponse{
		Status:     status,
		Timestamp:  time.Now().UTC(),
		Components: components,
	}

	statusCode := fasthttp.StatusOK
	if status == HealthStatusUnhealthy {
		statusCode = fasthttp.StatusServiceUnavailable
	}

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

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)

	body, err := json.Marshal(response)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode health check response")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetBody(body)
}

func (h *HealthHandler) checkComponents() []ComponentHealth {
	components := make([]ComponentHealth, 0, 3)

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
	producerHealthy := h.kafkaProducer != nil && h.kafkaProducer.IsHealthy()
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
	consumerHealthy := h.kafkaConsumer != nil && h.kafkaConsumer.IsHealthy()
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
