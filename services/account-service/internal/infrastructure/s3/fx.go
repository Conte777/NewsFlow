package s3

import (
	"context"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// Module provides S3/MinIO client for FX
var Module = fx.Module("s3",
	fx.Provide(newConfig),
	fx.Provide(NewClient),
	fx.Provide(provideMediaUploader),
	fx.Invoke(registerLifecycle),
)

// provideMediaUploader provides domain.MediaUploader interface from S3 Client
func provideMediaUploader(client *Client) domain.MediaUploader {
	return client
}

func newConfig(cfg *config.Config) *Config {
	return &Config{
		Endpoint:  cfg.S3.Endpoint,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		Bucket:    cfg.S3.Bucket,
		UseSSL:    cfg.S3.UseSSL,
		PublicURL: cfg.S3.PublicURL,
	}
}

type lifecycleParams struct {
	fx.In

	LC     fx.Lifecycle
	Client *Client
	Logger *zerolog.Logger
}

func registerLifecycle(p lifecycleParams) {
	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.Logger.Info().Msg("initializing S3/MinIO client...")
			if err := p.Client.EnsureBucket(ctx); err != nil {
				return err
			}
			p.Logger.Info().Msg("S3/MinIO client initialized successfully")
			return nil
		},
	})
}
