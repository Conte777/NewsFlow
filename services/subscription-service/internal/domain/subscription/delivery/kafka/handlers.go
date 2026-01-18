package kafka

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/deps"
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/dto"
	suberrors "github.com/Conte777/NewsFlow/services/subscription-service/internal/domain/subscription/errors"
	"github.com/rs/zerolog"
)

type EventHandler struct {
	usecase   deps.SubscriptionUseCase
	logger    zerolog.Logger
	processed uint64
	errors    uint64
}

func NewEventHandler(usecase deps.SubscriptionUseCase, logger zerolog.Logger) *EventHandler {
	return &EventHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *EventHandler) Handle(event *dto.SubscriptionEvent) error {
	ctx := context.Background()

	switch event.Type {
	// Saga: Subscription flow
	case dto.EventTypeSubscriptionRequested:
		return h.handleSubscriptionRequested(ctx, event)
	case dto.EventTypeSubscriptionActivated:
		return h.handleSubscriptionActivated(ctx, event)
	case dto.EventTypeSubscriptionFailed:
		return h.handleSubscriptionFailed(ctx, event)

	// Saga: Unsubscription flow
	case dto.EventTypeUnsubscriptionRequested:
		return h.handleUnsubscriptionRequested(ctx, event)
	case dto.EventTypeUnsubscriptionCompleted:
		return h.handleUnsubscriptionCompleted(ctx, event)
	case dto.EventTypeUnsubscriptionFailed:
		return h.handleUnsubscriptionFailed(ctx, event)

	default:
		h.logger.Warn().
			Str("event_type", event.Type).
			Msg("received unknown subscription event type")
		return nil
	}
}

// Saga: Subscription flow handlers

func (h *EventHandler) handleSubscriptionRequested(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Uint64("processed_total", atomic.LoadUint64(&h.processed)).
			Msg("subscription.requested event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" || event.ChannelName == "" {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("subscription.requested event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().Err(err).
			Str("user_id", event.UserID).
			Msg("invalid UserID in subscription.requested")
		return err
	}

	err = h.usecase.HandleSubscriptionRequested(ctx, userID, event.ChannelID, event.ChannelName)
	if err != nil {
		if errors.Is(err, suberrors.ErrSubscriptionAlreadyExists) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription already exists, skipping")
			return nil
		}
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription.requested handled successfully")

	return nil
}

func (h *EventHandler) handleSubscriptionActivated(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Msg("subscription.activated event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		h.logger.Error().
			Str("user_id", event.UserID).
			Str("channel_id", event.ChannelID).
			Msg("subscription.activated event missing required fields")
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	if err := h.usecase.HandleSubscriptionActivated(ctx, userID, event.ChannelID); err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("subscription activated successfully")

	return nil
}

func (h *EventHandler) handleSubscriptionFailed(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Msg("subscription.failed event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	// Extract reason from ChannelName field (used as reason in failure events)
	reason := event.ChannelName
	if reason == "" {
		reason = "Unknown error"
	}

	if err := h.usecase.HandleSubscriptionFailed(ctx, userID, event.ChannelID, reason); err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Str("reason", reason).
		Msg("subscription failure handled, rejection sent")

	return nil
}

// Saga: Unsubscription flow handlers

func (h *EventHandler) handleUnsubscriptionRequested(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Msg("unsubscription.requested event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	err = h.usecase.HandleUnsubscriptionRequested(ctx, userID, event.ChannelID)
	if err != nil {
		if errors.Is(err, suberrors.ErrSubscriptionNotFound) {
			h.logger.Warn().
				Str("user_id", event.UserID).
				Str("channel_id", event.ChannelID).
				Msg("subscription not found for unsubscription, skipping")
			return nil
		}
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("unsubscription.requested handled successfully")

	return nil
}

func (h *EventHandler) handleUnsubscriptionCompleted(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Msg("unsubscription.completed event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	if err := h.usecase.HandleUnsubscriptionCompleted(ctx, userID, event.ChannelID); err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Msg("unsubscription completed, subscription deleted")

	return nil
}

func (h *EventHandler) handleUnsubscriptionFailed(ctx context.Context, event *dto.SubscriptionEvent) error {
	start := time.Now()
	defer func() {
		atomic.AddUint64(&h.processed, 1)
		h.logger.Info().
			Dur("duration", time.Since(start)).
			Msg("unsubscription.failed event processed")
	}()

	if event.UserID == "" || event.ChannelID == "" {
		atomic.AddUint64(&h.errors, 1)
		return errors.New("missing required fields")
	}

	userID, err := strconv.ParseInt(event.UserID, 10, 64)
	if err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	reason := event.ChannelName
	if reason == "" {
		reason = "Unknown error"
	}

	if err := h.usecase.HandleUnsubscriptionFailed(ctx, userID, event.ChannelID, reason); err != nil {
		atomic.AddUint64(&h.errors, 1)
		return err
	}

	h.logger.Info().
		Str("user_id", event.UserID).
		Str("channel_id", event.ChannelID).
		Str("reason", reason).
		Msg("unsubscription failure handled, rejection sent")

	return nil
}
