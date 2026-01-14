package account

import (
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/domain/account/delivery/http"
	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/infrastructure/http/server"
	"go.uber.org/fx"
)

// Module provides account domain components for fx DI
var Module = fx.Module("account",
	fx.Provide(
		http.NewHealthHandler,
		http.NewRouter,
	),
	fx.Invoke(registerRoutes),
)

// registerRoutes registers account HTTP routes on the server
func registerRoutes(srv *server.Server, router *http.Router) {
	router.RegisterRoutes(srv.Router)
}
