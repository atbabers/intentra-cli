// Package debug provides debug logging utilities for the intentra CLI.
// Debug output is controlled by the debug config option or -d flag.
package debug

import (
	"fmt"
	"os"
	"time"
)

// Enabled controls whether debug logging is active.
var Enabled bool

// Log writes a debug message to stderr if debug mode is enabled.
func Log(format string, args ...any) {
	if Enabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// LogHTTP logs an HTTP request with method, URL, and status code.
func LogHTTP(method, url string, statusCode int) {
	if Enabled {
		ts := time.Now().UTC().Format(time.RFC3339)
		if statusCode == 0 {
			fmt.Fprintf(os.Stderr, "[DEBUG] %s %s %s -> (failed)\n", ts, method, url)
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] %s %s %s -> %d\n", ts, method, url, statusCode)
		}
	}
}

// Warn logs a warning message to stderr if debug mode is enabled.
func Warn(format string, args ...any) {
	if Enabled {
		fmt.Fprintf(os.Stderr, "[DEBUG] WARN: "+format+"\n", args...)
	}
}
