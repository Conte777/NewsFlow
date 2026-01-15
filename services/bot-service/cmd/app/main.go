package main

import (
	"go.uber.org/fx"

	"github.com/Conte777/NewsFlow/services/bot-service/internal/app"
)

func main() {
	fx.New(app.CreateApp()).Run()
}
