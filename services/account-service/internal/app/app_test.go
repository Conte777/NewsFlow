package app

import (
	"testing"

	"go.uber.org/fx"
)

func Test__CreateApp(t *testing.T) {
	t.Skip("Skipping fx validation test - requires full environment setup")

	if err := fx.ValidateApp(CreateApp()); err != nil {
		t.Errorf("fx validation failed: %v", err)
	}
}
