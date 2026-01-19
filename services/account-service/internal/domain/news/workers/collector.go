package workers

import (
	"context"
	"sync"
	"time"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/news/usecase/business"
	"github.com/rs/zerolog"
)

// CollectorWorker periodically collects news from subscribed channels
// When real-time updates are healthy, it acts as a fallback with longer intervals
// When real-time updates are unhealthy, it switches to normal polling mode
type CollectorWorker struct {
	newsUseCase    *business.UseCase
	accountManager domain.AccountManager
	interval       time.Duration // Normal polling interval (when updates are unhealthy)
	fallbackInt    time.Duration // Fallback interval (when updates are healthy)
	timeout        time.Duration
	logger         zerolog.Logger

	done   chan struct{}
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewCollectorWorker creates a new news collector worker
func NewCollectorWorker(
	newsUseCase *business.UseCase,
	accountManager domain.AccountManager,
	newsCfg *config.NewsConfig,
	logger zerolog.Logger,
) *CollectorWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &CollectorWorker{
		newsUseCase:    newsUseCase,
		accountManager: accountManager,
		interval:       newsCfg.PollInterval,
		fallbackInt:    newsCfg.FallbackPollInterval,
		timeout:        newsCfg.CollectionTimeout,
		logger:         logger,
		done:           make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts the news collection worker
func (w *CollectorWorker) Start() {
	w.logger.Info().
		Dur("interval", w.interval).
		Dur("fallback_interval", w.fallbackInt).
		Dur("timeout", w.timeout).
		Msg("Starting news collector worker (fallback mode when real-time updates are healthy)")

	w.wg.Add(1)
	go w.run()
}

// Stop gracefully stops the news collection worker
func (w *CollectorWorker) Stop() {
	w.logger.Info().Msg("Stopping news collector worker")

	w.cancel()
	close(w.done)
	w.wg.Wait()

	w.logger.Info().Msg("News collector worker stopped")
}

// areUpdatesHealthy checks if any account has healthy real-time updates
func (w *CollectorWorker) areUpdatesHealthy() bool {
	accounts := w.accountManager.GetAllAccounts()
	for _, acc := range accounts {
		if acc.IsConnected() && acc.IsUpdatesHealthy() {
			return true
		}
	}
	return false
}

// run is the main worker loop with dynamic interval based on updates health
func (w *CollectorWorker) run() {
	defer w.wg.Done()

	// Start with normal interval, will adjust based on updates health
	currentInterval := w.interval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			// Check updates health and adjust interval
			updatesHealthy := w.areUpdatesHealthy()
			newInterval := w.interval
			if updatesHealthy {
				newInterval = w.fallbackInt
			}

			// Adjust ticker if interval changed
			if newInterval != currentInterval {
				currentInterval = newInterval
				ticker.Reset(currentInterval)
				if updatesHealthy {
					w.logger.Info().
						Dur("new_interval", currentInterval).
						Msg("Real-time updates healthy, switching to fallback polling interval")
				} else {
					w.logger.Info().
						Dur("new_interval", currentInterval).
						Msg("Real-time updates unhealthy, switching to normal polling interval")
				}
			}

			// Always collect news as fallback, even if updates are healthy
			// This catches any missed messages and ensures consistency
			w.collectNews(updatesHealthy)
		}
	}
}

// collectNews performs a single news collection cycle
// isFallback indicates if this is a fallback collection when updates are healthy
func (w *CollectorWorker) collectNews(isFallback bool) {
	ctx, cancel := context.WithTimeout(w.ctx, w.timeout)
	defer cancel()

	if isFallback {
		w.logger.Debug().Msg("Starting fallback news collection cycle")
	} else {
		w.logger.Debug().Msg("Starting news collection cycle (updates unhealthy)")
	}

	if err := w.newsUseCase.CollectNews(ctx); err != nil {
		if ctx.Err() != nil {
			w.logger.Warn().Err(err).Msg("News collection cancelled or timed out")
		} else {
			w.logger.Error().Err(err).Msg("News collection failed")
		}
		return
	}

	if isFallback {
		w.logger.Debug().Msg("Fallback news collection cycle completed")
	} else {
		w.logger.Debug().Msg("News collection cycle completed")
	}
}
