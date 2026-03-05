// Package httputil provides shared HTTP constants and clients used across
// multiple packages to avoid duplication.
package httputil

import (
	"net/http"
	"time"
)

// MaxResponseSize is the maximum allowed HTTP response body size (10 MB).
const MaxResponseSize = 10 * 1024 * 1024

// DefaultClient is the shared HTTP client for operations requiring a 30s timeout.
var DefaultClient = &http.Client{Timeout: 30 * time.Second}
