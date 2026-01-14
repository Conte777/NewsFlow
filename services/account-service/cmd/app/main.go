package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/YarosTrubechkoi/telegram-news-feed/account-service/internal/app"
	"go.uber.org/fx"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fxApp := fx.New(app.CreateApp())

	if err := fxApp.Start(ctx); err != nil {
		panic(err)
	}

	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := fxApp.Stop(stopCtx); err != nil {
		panic(err)
	}
}
