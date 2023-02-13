package integrationtest

import (
	"github.com/caddyserver/caddy/v2/caddytest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// NewCaddyTester creates a new tester. Relative to this method here
func NewCaddyTester(t *testing.T) *caddytest.Tester {
	_, file, _, _ := runtime.Caller(0)

	// file always points to "caddyhelper.go" (this file)
	os.Chdir(filepath.Dir(file))
	return caddytest.NewTester(t)
}
