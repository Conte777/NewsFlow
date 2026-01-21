package workers

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	channeldeps "github.com/Conte777/NewsFlow/services/account-service/internal/domain/channel/deps"
)

// DeletionChecker periodically checks if recently cached messages were deleted
// This is a fallback mechanism for when real-time deletion events are missed
// due to gotd/td pts filtering
type DeletionChecker struct {
	accountManager    domain.AccountManager
	channelRepo       channeldeps.ChannelRepository
	recentMsgCache    channeldeps.RecentMessagesCache
	kafkaProducer     domain.KafkaProducer
	logger            zerolog.Logger
	interval          time.Duration
	lookbackCount     int
	timeout           time.Duration

	done   chan struct{}
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewDeletionChecker creates a new deletion checker worker
func NewDeletionChecker(
	accountManager domain.AccountManager,
	channelRepo channeldeps.ChannelRepository,
	recentMsgCache channeldeps.RecentMessagesCache,
	kafkaProducer domain.KafkaProducer,
	newsCfg *config.NewsConfig,
	logger zerolog.Logger,
) *DeletionChecker {
	ctx, cancel := context.WithCancel(context.Background())

	interval := 30 * time.Second
	lookbackCount := 100
	timeout := 60 * time.Second

	if newsCfg != nil {
		if newsCfg.DeletionCheckInterval > 0 {
			interval = newsCfg.DeletionCheckInterval
		}
		if newsCfg.DeletionCheckLookback > 0 {
			lookbackCount = newsCfg.DeletionCheckLookback
		}
	}

	return &DeletionChecker{
		accountManager:    accountManager,
		channelRepo:       channelRepo,
		recentMsgCache:    recentMsgCache,
		kafkaProducer:     kafkaProducer,
		logger:            logger.With().Str("component", "deletion_checker").Logger(),
		interval:          interval,
		lookbackCount:     lookbackCount,
		timeout:           timeout,
		done:              make(chan struct{}),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start starts the deletion checker worker
func (c *DeletionChecker) Start() {
	c.logger.Info().
		Dur("interval", c.interval).
		Int("lookback_count", c.lookbackCount).
		Msg("starting deletion checker worker")

	c.wg.Add(1)
	go c.run()
}

// Stop gracefully stops the deletion checker worker
func (c *DeletionChecker) Stop() {
	c.logger.Info().Msg("stopping deletion checker worker")

	c.cancel()
	close(c.done)
	c.wg.Wait()

	c.logger.Info().Msg("deletion checker worker stopped")
}

// run is the main worker loop
func (c *DeletionChecker) run() {
	defer c.wg.Done()

	// Initial delay to let other components start
	select {
	case <-c.done:
		return
	case <-time.After(5 * time.Second):
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.checkDeletedMessages()
		}
	}
}

// checkDeletedMessages checks all cached messages for deletions
func (c *DeletionChecker) checkDeletedMessages() {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	channelIDs := c.recentMsgCache.GetAllChannelIDs()
	if len(channelIDs) == 0 {
		c.logger.Debug().Msg("no channels with cached messages to check")
		return
	}

	c.logger.Debug().
		Int("channels_count", len(channelIDs)).
		Msg("starting deletion check cycle")

	var totalDeleted int

	for _, channelID := range channelIDs {
		select {
		case <-ctx.Done():
			c.logger.Warn().Msg("deletion check cycle cancelled")
			return
		default:
		}

		deleted := c.checkChannelDeletions(ctx, channelID)
		totalDeleted += deleted
	}

	if totalDeleted > 0 {
		c.logger.Info().
			Int("total_deleted", totalDeleted).
			Int("channels_checked", len(channelIDs)).
			Msg("deletion check cycle completed with findings")
	} else {
		c.logger.Debug().
			Int("channels_checked", len(channelIDs)).
			Msg("deletion check cycle completed")
	}
}

// checkChannelDeletions checks for deleted messages in a specific channel
func (c *DeletionChecker) checkChannelDeletions(ctx context.Context, channelID string) int {
	// Get recent message IDs from cache
	messageIDs := c.recentMsgCache.GetRecent(channelID, c.lookbackCount)
	if len(messageIDs) == 0 {
		return 0
	}

	// Get an available account
	account, err := c.accountManager.GetAvailableAccount()
	if err != nil {
		c.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Msg("no available account for deletion check")
		return 0
	}

	// Check which messages still exist
	existingIDs, err := account.CheckMessagesExist(ctx, channelID, messageIDs)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("channel_id", channelID).
			Msg("failed to check message existence")
		return 0
	}

	// Find deleted messages (in cache but not in Telegram)
	deletedIDs := c.findDeleted(messageIDs, existingIDs)
	if len(deletedIDs) == 0 {
		return 0
	}

	c.logger.Info().
		Str("channel_id", channelID).
		Ints("deleted_ids", deletedIDs).
		Int("checked", len(messageIDs)).
		Int("existing", len(existingIDs)).
		Msg("found deleted messages via fallback check")

	// Send deletion event to Kafka
	if err := c.kafkaProducer.SendNewsDeleted(ctx, channelID, deletedIDs); err != nil {
		c.logger.Error().Err(err).
			Str("channel_id", channelID).
			Ints("deleted_ids", deletedIDs).
			Msg("failed to send news deleted event to Kafka")
		return 0
	}

	// Remove deleted messages from cache
	c.recentMsgCache.Remove(channelID, deletedIDs)

	return len(deletedIDs)
}

// findDeleted returns message IDs that are in cached but not in existing
func (c *DeletionChecker) findDeleted(cached, existing []int) []int {
	existingSet := make(map[int]struct{}, len(existing))
	for _, id := range existing {
		existingSet[id] = struct{}{}
	}

	var deleted []int
	for _, id := range cached {
		if _, found := existingSet[id]; !found {
			deleted = append(deleted, id)
		}
	}

	return deleted
}
