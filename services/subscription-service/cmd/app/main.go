package main

import (
	"github.com/Conte777/NewsFlow/services/subscription-service/internal/app"
	"go.uber.org/fx"
)

func main() {
	fx.New(app.CreateApp()).Run()
}
