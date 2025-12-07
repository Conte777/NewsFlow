package kafka

import (
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain"

	"github.com/rs/zerolog"
)

type SubscriptionEventHandler struct {
	usecase domain.SubscriptionUseCase
	logger  zerolog.Logger
}

func NewSubscriptionEventHandler(
	usecase domain.SubscriptionUseCase,
	logger zerolog.Logger,
) *SubscriptionEventHandler {
	return &SubscriptionEventHandler{
		usecase: usecase,
		logger:  logger,
	}
}
