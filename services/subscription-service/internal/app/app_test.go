package app

import (
	"testing"

	"go.uber.org/fx"
)

func TestCreateApp(t *testing.T) {
	err := fx.ValidateApp(CreateApp())
	if err != nil {
		t.Fatalf("fx validation failed: %v", err)
	}
}
