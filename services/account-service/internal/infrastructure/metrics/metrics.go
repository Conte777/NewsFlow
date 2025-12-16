package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the account service
type Metrics struct {
	// Channel subscription metrics
	SubscriptionsTotal       prometheus.Counter
	SubscriptionErrors       *prometheus.CounterVec
	UnsubscriptionsTotal     prometheus.Counter
	UnsubscriptionErrors     *prometheus.CounterVec
	ActiveSubscriptions      prometheus.Gauge

	// News collection metrics
	NewsCollectionTotal      prometheus.Counter
	NewsCollectionErrors     prometheus.Counter
	NewsItemsCollected       prometheus.Counter
	NewsCollectionDuration   prometheus.Histogram

	// Account metrics
	ActiveAccounts           prometheus.Gauge
	TotalAccounts            prometheus.Gauge
	AccountReconnections     prometheus.Counter
	AccountRateLimits        prometheus.Counter

	// Kafka metrics
	KafkaMessagesProduced    prometheus.Counter
	KafkaProduceErrors       *prometheus.CounterVec
	KafkaProduceDuration     prometheus.Histogram

	// Operation duration metrics
	SubscriptionDuration     prometheus.Histogram
	UnsubscriptionDuration   prometheus.Histogram
}

var (
	// DefaultMetrics is the default metrics instance
	DefaultMetrics *Metrics
	once           sync.Once
)

// GetDefaultMetrics returns the singleton metrics instance
func GetDefaultMetrics() *Metrics {
	once.Do(func() {
		DefaultMetrics = NewMetrics()
	})
	return DefaultMetrics
}

func init() {
	// Initialize DefaultMetrics on package import
	GetDefaultMetrics()
}

// NewMetrics creates a new Metrics instance with all counters and gauges
func NewMetrics() *Metrics {
	return &Metrics{
		// Channel subscription metrics
		SubscriptionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_subscriptions_total",
			Help: "Total number of channel subscriptions created",
		}),
		SubscriptionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "account_service_subscription_errors_total",
				Help: "Total number of subscription errors",
			},
			[]string{"error_type"},
		),
		UnsubscriptionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_unsubscriptions_total",
			Help: "Total number of channel unsubscriptions",
		}),
		UnsubscriptionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "account_service_unsubscription_errors_total",
				Help: "Total number of unsubscription errors",
			},
			[]string{"error_type"},
		),
		ActiveSubscriptions: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "account_service_active_subscriptions",
			Help: "Current number of active channel subscriptions",
		}),

		// News collection metrics
		NewsCollectionTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_news_collections_total",
			Help: "Total number of news collection cycles",
		}),
		NewsCollectionErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_news_collection_errors_total",
			Help: "Total number of news collection errors",
		}),
		NewsItemsCollected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_news_items_sent_total",
			Help: "Total number of news items sent to Kafka",
		}),
		NewsCollectionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "account_service_news_collection_duration_seconds",
			Help:    "Duration of news collection cycles in seconds",
			Buckets: []float64{0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		}),

		// Account metrics
		ActiveAccounts: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "account_service_active_accounts",
			Help: "Current number of active Telegram accounts",
		}),
		TotalAccounts: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "account_service_total_accounts",
			Help: "Total number of configured Telegram accounts",
		}),
		AccountReconnections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_account_reconnections_total",
			Help: "Total number of Telegram account reconnections",
		}),
		AccountRateLimits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_account_rate_limits_total",
			Help: "Total number of rate limit events from Telegram API",
		}),

		// Kafka metrics
		KafkaMessagesProduced: promauto.NewCounter(prometheus.CounterOpts{
			Name: "account_service_kafka_messages_produced_total",
			Help: "Total number of messages produced to Kafka",
		}),
		KafkaProduceErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "account_service_kafka_produce_errors_total",
				Help: "Total number of Kafka produce errors",
			},
			[]string{"error_type"},
		),
		KafkaProduceDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "account_service_kafka_produce_duration_seconds",
			Help:    "Duration of Kafka produce operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}),

		// Operation duration metrics
		SubscriptionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "account_service_subscription_duration_seconds",
			Help:    "Duration of subscription operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		}),
		UnsubscriptionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "account_service_unsubscription_duration_seconds",
			Help:    "Duration of unsubscription operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		}),
	}
}

// RecordSubscription records a successful subscription
func (m *Metrics) RecordSubscription(duration float64) {
	m.SubscriptionsTotal.Inc()
	m.SubscriptionDuration.Observe(duration)
}

// RecordSubscriptionError records a subscription error with error type
func (m *Metrics) RecordSubscriptionError(errorType string) {
	if errorType == "" {
		errorType = "unknown"
	}
	m.SubscriptionErrors.WithLabelValues(errorType).Inc()
}

// RecordUnsubscription records a successful unsubscription
func (m *Metrics) RecordUnsubscription(duration float64) {
	m.UnsubscriptionsTotal.Inc()
	m.UnsubscriptionDuration.Observe(duration)
}

// RecordUnsubscriptionError records an unsubscription error with error type
func (m *Metrics) RecordUnsubscriptionError(errorType string) {
	if errorType == "" {
		errorType = "unknown"
	}
	m.UnsubscriptionErrors.WithLabelValues(errorType).Inc()
}

// RecordNewsCollection records a news collection cycle
func (m *Metrics) RecordNewsCollection(itemsSent int, duration float64) {
	m.NewsCollectionTotal.Inc()
	// Only add positive values to prevent counter from going backwards
	if itemsSent > 0 {
		m.NewsItemsCollected.Add(float64(itemsSent))
	}
	m.NewsCollectionDuration.Observe(duration)
}

// RecordNewsCollectionError records a news collection error
func (m *Metrics) RecordNewsCollectionError() {
	m.NewsCollectionErrors.Inc()
}

// UpdateActiveSubscriptions updates the active subscriptions gauge
func (m *Metrics) UpdateActiveSubscriptions(count int) {
	m.ActiveSubscriptions.Set(float64(count))
}

// UpdateAccounts updates account metrics
func (m *Metrics) UpdateAccounts(active, total int) {
	m.ActiveAccounts.Set(float64(active))
	m.TotalAccounts.Set(float64(total))
}

// RecordAccountReconnection records a Telegram account reconnection
func (m *Metrics) RecordAccountReconnection() {
	m.AccountReconnections.Inc()
}

// RecordAccountRateLimit records a rate limit event from Telegram API
func (m *Metrics) RecordAccountRateLimit() {
	m.AccountRateLimits.Inc()
}

// RecordKafkaMessage records a Kafka message production with duration
func (m *Metrics) RecordKafkaMessage(duration float64) {
	m.KafkaMessagesProduced.Inc()
	m.KafkaProduceDuration.Observe(duration)
}

// RecordKafkaError records a Kafka production error with error type
func (m *Metrics) RecordKafkaError(errorType string) {
	if errorType == "" {
		errorType = "unknown"
	}
	m.KafkaProduceErrors.WithLabelValues(errorType).Inc()
}
