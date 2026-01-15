package qrauth

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/Conte777/NewsFlow/services/account-service/config"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain"
	qrhttp "github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/delivery/http"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/deps"
	"github.com/Conte777/NewsFlow/services/account-service/internal/domain/qrauth/usecase/business"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/http/server"
	"github.com/Conte777/NewsFlow/services/account-service/internal/infrastructure/telegram"
)

// Module provides QR authentication components for fx DI
var Module = fx.Module("qrauth",
	fx.Provide(NewQRSessionStoreFx),
	fx.Provide(NewQRAuthManagerFx),
	fx.Provide(NewQRAuthUseCaseFx),
	fx.Provide(NewQRAuthHandlerFx),
	fx.Provide(NewQRAuthRouterFx),
	fx.Invoke(RegisterRoutes),
)

// NewQRSessionStoreFx creates a QR session store for fx DI
func NewQRSessionStoreFx(lc fx.Lifecycle, logger zerolog.Logger) *telegram.QRSessionStore {
	store := telegram.NewQRSessionStore(
		5*time.Minute,  // sessionTTL
		1*time.Minute,  // cleanupInterval
		100,            // maxSessions
		logger,
	)

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			store.Stop()
			return nil
		},
	})

	return store
}

// NewQRAuthManagerFx creates a QR auth manager for fx DI
func NewQRAuthManagerFx(
	db *gorm.DB,
	telegramCfg *config.TelegramConfig,
	accountManager domain.AccountManager,
	sessionStore *telegram.QRSessionStore,
	logger zerolog.Logger,
) *telegram.QRAuthManager {
	return telegram.NewQRAuthManager(db, telegramCfg, accountManager, sessionStore, logger)
}

// NewQRAuthUseCaseFx creates a QR auth use case for fx DI
func NewQRAuthUseCaseFx(manager *telegram.QRAuthManager, logger zerolog.Logger) deps.QRAuthService {
	return business.NewQRAuthUseCase(manager, logger)
}

// NewQRAuthHandlerFx creates a QR auth handler for fx DI
func NewQRAuthHandlerFx(useCase deps.QRAuthService, logger zerolog.Logger) *qrhttp.QRAuthHandler {
	return qrhttp.NewQRAuthHandler(useCase, logger)
}

// NewQRAuthRouterFx creates a QR auth router for fx DI
func NewQRAuthRouterFx(handler *qrhttp.QRAuthHandler, logger zerolog.Logger) *qrhttp.Router {
	return qrhttp.NewRouter(handler, logger)
}

// RegisterRoutes registers QR auth routes on the server
func RegisterRoutes(server *server.Server, router *qrhttp.Router) {
	router.RegisterRoutes(server.Router)
}
