package debug

import (
	"testing"
)

func TestLog_WhenDisabled(t *testing.T) {
	Enabled = false
	Log("should not panic: %s", "test")
}

func TestLog_WhenEnabled(t *testing.T) {
	Enabled = true
	defer func() { Enabled = false }()
	Log("test message: %s", "value")
}

func TestLogHTTP_WhenDisabled(t *testing.T) {
	Enabled = false
	LogHTTP("GET", "http://example.com", 200)
	LogHTTP("POST", "http://example.com", 0)
}

func TestLogHTTP_WhenEnabled(t *testing.T) {
	Enabled = true
	defer func() { Enabled = false }()
	LogHTTP("GET", "http://example.com", 200)
	LogHTTP("POST", "http://example.com", 0)
}

func TestWarn_WhenDisabled(t *testing.T) {
	Enabled = false
	Warn("should not panic: %d", 123)
}

func TestWarn_WhenEnabled(t *testing.T) {
	Enabled = true
	defer func() { Enabled = false }()
	Warn("warning message: %d", 456)
}
