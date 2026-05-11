package runtime

import (
	"testing"

	"terraform-runner/internal/config"
)

func TestNewBuildsRuntime(t *testing.T) {
	app := New(config.Config{})
	if app == nil {
		t.Fatal("expected runtime app")
	}
}
