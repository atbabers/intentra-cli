package device

import (
	"testing"
)

func TestGetDeviceID(t *testing.T) {
	id, err := GetDeviceID()
	if err != nil {
		t.Fatalf("GetDeviceID failed: %v", err)
	}
	if id == "" {
		t.Error("GetDeviceID returned empty string")
	}
	if len(id) != 32 {
		t.Errorf("Expected device ID length 32, got %d", len(id))
	}
}

func TestGetDeviceID_Deterministic(t *testing.T) {
	id1, err := GetDeviceID()
	if err != nil {
		t.Fatalf("First GetDeviceID failed: %v", err)
	}

	id2, err := GetDeviceID()
	if err != nil {
		t.Fatalf("Second GetDeviceID failed: %v", err)
	}

	if id1 != id2 {
		t.Errorf("Device ID not deterministic: %s != %s", id1, id2)
	}
}

func TestGetMetadata(t *testing.T) {
	meta := GetMetadata()

	if meta.Platform == "" {
		t.Error("Platform should not be empty")
	}
}

func TestGetFallbackID(t *testing.T) {
	id, err := getFallbackID()
	if err != nil {
		t.Fatalf("getFallbackID failed: %v", err)
	}
	if id == "" {
		t.Error("Fallback ID should not be empty")
	}
}
