package kafka

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/dto"
	suberrors "github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/errors"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/usecase/buissines"
	"github.com/rs/zerolog"
)

type EventHandler struct {
	usecase   *buissines.UseCase
	logger    zerolog.Logger
	processed uint64
	errors    uint64
}

func NewEventHandler(usecase *buissines.UseCase, logger zerolog.Logger) *EventHandler {
	return &EventHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *EventHandler) Handle(event *dto.SubscriptionEvent) error {
	ctx := context.Background()

	switch event.Type {
	case "subscription_created":
		return h.handleCreated(ctx, event)
	case "subscription_cancelled":
		return h.handleDeleted(ctx, event)
	default:
		h.logger.Warn().
			Str("event_type", event.Type).
			Msg("received unknown subscription event type")
		return nil
	}
}

func (h *EventHandler) handleCreated(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Uint64("processed_total", atomic.LoadUint64(&h.processed)).
			Uint64("errors_total", atomic.LoadUint64(&h.errors)).
			Msg("subscription created event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" || event.ChannelName == "" {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Str("channel_name", event.ChannelName).
			Msg("subscription created event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	err = h.usecase.CreateSubscription(ctx, userID, event.ChannelID, event.ChannelName)
	if err != nil {
		if errors.Is(err, suberrors.ErrSubscriptionAlreadyExists) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription already exists, skipping creation")
			return nil
		}
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully created")

	return nil
}

func (h *EventHandler) handleDeleted(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Uint64("processed_total", atomic.LoadUint64(&h.processed)).
			Uint64("errors_total", atomic.LoadUint64(&h.errors)).
			Msg("subscription deleted event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("subscription deleted event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID, cannot convert to int64")
		return err
	}

	err = h.usecase.DeleteSubscription(ctx, userID, event.ChannelID)
	if err != nil {
		if errors.Is(err, suberrors.ErrSubscriptionNotFound) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription does not exist, skipping deletion")
			return nil
		}
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription successfully deleted")

	return nil
}
