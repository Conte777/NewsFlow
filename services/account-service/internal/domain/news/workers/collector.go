package workers

import (
	"context"
	"sync"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/config"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/news/usecase/business"
	"github.com/rs/zerolog"
)

// CollectorWorker periodically collects news from subscribed channels
type CollectorWorker struct {
	newsUseCase *business.UseCase
	interval    time.Duration
	timeout     time.Duration
	logger      zerolog.Logger

	done   chan struct{}
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewCollectorWorker creates a new news collector worker
func NewCollectorWorker(
	newsUseCase *business.UseCase,
	newsCfg *config.NewsConfig,
	logger zerolog.Logger,
) *CollectorWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &CollectorWorker{
		newsUseCase: newsUseCase,
		interval:    newsCfg.PollInterval,
		timeout:     newsCfg.CollectionTimeout,
		logger:      logger,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the news collection worker
func (w *CollectorWorker) Start() {
	w.logger.Info().
		Dur("interval", w.interval).
		Dur("timeout", w.timeout).
		Msg("Starting news collector worker")

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

// run is the main worker loop
func (w *CollectorWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			w.collectNews()
		}
	}
}

// collectNews performs a single news collection cycle
func (w *CollectorWorker) collectNews() {
	ctx, cancel := context.WithTimeout(w.ctx, w.timeout)
	defer cancel()

	w.logger.Debug().Msg("Starting news collection cycle")

	if err := w.newsUseCase.CollectNews(ctx); err != nil {
		if ctx.Err() != nil {
			w.logger.Warn().Err(err).Msg("News collection cancelled or timed out")
		} else {
			w.logger.Error().Err(err).Msg("News collection failed")
		}
		return
	}

	w.logger.Debug().Msg("News collection cycle completed")
}
