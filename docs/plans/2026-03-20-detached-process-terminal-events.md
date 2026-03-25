# Detached Process for Terminal Event Network I/O

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate "Hook cancelled" errors by ensuring terminal event handlers never block on network I/O — fork a detached child process for HTTP calls instead.

**Architecture:** When a terminal event (Stop, SessionEnd) requires network I/O, the handler writes necessary data to a temp file, then spawns `intentra __send` as a detached child process. The parent returns immediately (satisfying the AI tool's hook timeout). The child reads the temp file, performs the HTTP call, and exits. This replaces inline HTTP calls in both `handleStopEvent` and `handleSessionEndEvent`.

**Tech Stack:** Go stdlib (`os/exec`, `syscall`, `encoding/json`), cobra subcommand

---

### Task 1: Add the hidden `__send` subcommand skeleton

**Files:**
- Create: `cmd/intentra/send.go`
- Modify: `cmd/intentra/main.go:55-64` (register new command)

**Step 1: Write the failing test for the new subcommand**

Create a test that verifies the `__send` command exists and accepts the expected flags.

```go
// cmd/intentra/send_test.go
package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestSendSubcommandExists(t *testing.T) {
	if os.Getenv("INTENTRA_BINARY") == "" {
		t.Skip("INTENTRA_BINARY not set, skipping integration test")
	}

	cmd := exec.Command(os.Getenv("INTENTRA_BINARY"), "__send", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("__send subcommand should exist: %v\nOutput: %s", err, out)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./cmd/intentra/ -run TestSendSubcommandExists -v`
Expected: SKIP (no binary set) — that's fine, this is an integration test. The unit tests come next.

**Step 3: Create the `__send` subcommand**

```go
// cmd/intentra/send.go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/intentrahq/intentra-cli/internal/api"
	"github.com/intentrahq/intentra-cli/internal/auth"
	"github.com/intentrahq/intentra-cli/internal/config"
	"github.com/intentrahq/intentra-cli/internal/debug"
	"github.com/intentrahq/intentra-cli/internal/queue"
	"github.com/intentrahq/intentra-cli/pkg/models"
	"github.com/spf13/cobra"
)

// sendPayload is the on-disk format for deferred send operations.
type sendPayload struct {
	Action     string      `json:"action"`
	Scan       *models.Scan `json:"scan,omitempty"`
	ScanID     string      `json:"scan_id,omitempty"`
	SessionKey string      `json:"session_key,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	DurationMs int64       `json:"duration_ms,omitempty"`
}

func newSendCmd() *cobra.Command {
	var payloadFile string

	cmd := &cobra.Command{
		Use:           "__send",
		Short:         "Send deferred hook data (internal use)",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeferredSend(payloadFile)
		},
	}

	cmd.Flags().StringVar(&payloadFile, "payload", "", "path to JSON payload file")
	return cmd
}

func runDeferredSend(payloadFile string) error {
	if payloadFile == "" {
		return fmt.Errorf("--payload is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	debug.Enabled = cfg.Debug

	data, err := os.ReadFile(payloadFile)
	if err != nil {
		return fmt.Errorf("failed to read payload: %w", err)
	}

	// Always clean up the payload file
	defer os.Remove(payloadFile)

	var payload sendPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	switch payload.Action {
	case "send_scan":
		return deferredSendScan(payload, cfg)
	case "patch_session_end":
		return deferredPatchSessionEnd(payload)
	default:
		return fmt.Errorf("unknown action: %s", payload.Action)
	}
}

func deferredSendScan(payload sendPayload, cfg *config.Config) error {
	scan := payload.Scan
	if scan == nil {
		return fmt.Errorf("send_scan payload missing scan")
	}

	synced := false

	creds, credErr := auth.GetValidCredentials()
	if credErr != nil {
		debug.Warn("credential check failed: %v", credErr)
	}
	if creds != nil {
		if err := api.SendScanWithJWT(scan, creds.AccessToken); err != nil {
			debug.Warn("failed to sync to api.intentra.sh: %v", err)
		} else {
			debug.Log("Synced to https://api.intentra.sh")
			synced = true
		}
	}

	if !synced && cfg.Server.Enabled {
		client, err := api.NewClient(cfg)
		if err == nil {
			if err := client.SendScan(scan); err != nil {
				debug.Warn("sync failed: %v", err)
			} else {
				synced = true
			}
		}
	}

	if !synced {
		if err := queue.Enqueue(scan); err != nil {
			debug.Warn("failed to queue scan offline: %v", err)
		}
	}

	if synced && creds != nil {
		queue.FlushWithJWT(creds.AccessToken)
	}

	return nil
}

func deferredPatchSessionEnd(payload sendPayload) error {
	if payload.ScanID == "" {
		return fmt.Errorf("patch_session_end payload missing scan_id")
	}

	creds, err := auth.GetValidCredentials()
	if err != nil {
		debug.Warn("credential check failed: %v", err)
		return nil
	}
	if creds == nil {
		debug.Log("no valid credentials for deferred patch, ignoring")
		return nil
	}

	if err := api.PatchSessionEnd(payload.ScanID, creds.AccessToken, payload.Reason, payload.DurationMs); err != nil {
		debug.Warn("deferred PATCH session end failed: %v", err)
	} else {
		debug.Log("Deferred PATCHed session end for scan %s", payload.ScanID)
	}

	return nil
}
```

**Step 4: Register the command in main.go**

In `cmd/intentra/main.go`, add after line 64 (after `newExtensionInfoCmd()`):

```go
	rootCmd.AddCommand(newSendCmd())
```

**Step 5: Verify it compiles**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go build ./cmd/intentra/`
Expected: Compiles with no errors.

**Step 6: Commit**

```bash
git add cmd/intentra/send.go cmd/intentra/send_test.go cmd/intentra/main.go
git commit -m "feat: add hidden __send subcommand for deferred network I/O"
```

---

### Task 2: Add `SpawnDetachedSend` helper in handler.go

**Files:**
- Modify: `internal/hooks/handler.go` (add new function)
- Create: `internal/hooks/handler_deferred_test.go`

**Step 1: Write the failing test**

```go
// internal/hooks/handler_deferred_test.go
package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentrahq/intentra-cli/pkg/models"
)

func TestWriteSendPayload_SendScan(t *testing.T) {
	scan := &models.Scan{
		ID:             "scan_test123",
		Tool:           "claude",
		ConversationID: "conv-1",
	}

	path, err := writeSendPayload("send_scan", scan, "", "", "", 0)
	if err != nil {
		t.Fatalf("writeSendPayload failed: %v", err)
	}
	defer os.Remove(path)

	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read payload file: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	if payload["action"] != "send_scan" {
		t.Errorf("expected action send_scan, got: %v", payload["action"])
	}
	if payload["scan"] == nil {
		t.Error("expected scan in payload")
	}
}

func TestWriteSendPayload_PatchSessionEnd(t *testing.T) {
	path, err := writeSendPayload("patch_session_end", nil, "scan_abc", "key1", "user_exit", 45000)
	if err != nil {
		t.Fatalf("writeSendPayload failed: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read payload file: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	if payload["action"] != "patch_session_end" {
		t.Errorf("expected action patch_session_end, got: %v", payload["action"])
	}
	if payload["scan_id"] != "scan_abc" {
		t.Errorf("expected scan_id scan_abc, got: %v", payload["scan_id"])
	}
	if payload["reason"] != "user_exit" {
		t.Errorf("expected reason user_exit, got: %v", payload["reason"])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -run TestWriteSendPayload -v`
Expected: FAIL — `writeSendPayload` undefined.

**Step 3: Implement `writeSendPayload` and `spawnDetachedSend`**

Add to the bottom of `internal/hooks/handler.go` (before the closing of the file):

```go
// sendPayload is the on-disk format passed to the __send subcommand.
type sendPayload struct {
	Action     string       `json:"action"`
	Scan       *models.Scan `json:"scan,omitempty"`
	ScanID     string       `json:"scan_id,omitempty"`
	SessionKey string       `json:"session_key,omitempty"`
	Reason     string       `json:"reason,omitempty"`
	DurationMs int64        `json:"duration_ms,omitempty"`
}

// writeSendPayload writes a deferred send payload to a temp file and returns its path.
func writeSendPayload(action string, scan *models.Scan, scanID, sessionKey, reason string, durationMs int64) (string, error) {
	payload := sendPayload{
		Action:     action,
		Scan:       scan,
		ScanID:     scanID,
		SessionKey: sessionKey,
		Reason:     reason,
		DurationMs: durationMs,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	f, err := os.CreateTemp("", "intentra_send_*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to write payload: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to close payload file: %w", err)
	}

	return f.Name(), nil
}

// spawnDetachedSend launches `intentra __send --payload <file>` as a detached process.
// The parent returns immediately; the child handles the network I/O.
func spawnDetachedSend(payloadPath string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find executable: %w", err)
	}

	cmd := exec.Command(exe, "__send", "--payload", payloadPath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = detachedProcAttr()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn deferred send: %w", err)
	}

	// Detach — do not wait for child
	go cmd.Wait()

	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -run TestWriteSendPayload -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/hooks/handler.go internal/hooks/handler_deferred_test.go
git commit -m "feat: add writeSendPayload and spawnDetachedSend helpers"
```

---

### Task 3: Add platform-specific `detachedProcAttr()`

The `SysProcAttr` fields for detaching a child differ between Unix and Windows. Use build tags.

**Files:**
- Create: `internal/hooks/proc_unix.go`
- Create: `internal/hooks/proc_windows.go`

**Step 1: Create Unix implementation**

```go
// internal/hooks/proc_unix.go
//go:build !windows

package hooks

import "syscall"

// detachedProcAttr returns SysProcAttr that creates a new session (setsid),
// detaching the child from the parent's process group so it survives parent exit.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
```

**Step 2: Create Windows implementation**

```go
// internal/hooks/proc_windows.go
//go:build windows

package hooks

import "syscall"

// detachedProcAttr returns SysProcAttr for Windows detached process creation.
// CREATE_NEW_PROCESS_GROUP detaches the child from the parent's console group.
const createNewProcessGroup = 0x00000200

func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}
```

**Step 3: Verify it compiles**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go build ./internal/hooks/`
Expected: Compiles with no errors.

**Step 4: Commit**

```bash
git add internal/hooks/proc_unix.go internal/hooks/proc_windows.go
git commit -m "feat: add platform-specific detached process attributes"
```

---

### Task 4: Rewrite `handleStopEvent` to use detached send

**Files:**
- Modify: `internal/hooks/handler.go:1005-1077` (`handleStopEvent`)

**Step 1: Write a test that verifies stop events write a payload file instead of sending inline**

```go
// Add to internal/hooks/handler_deferred_test.go

func TestHandleStopEvent_CreatesPayloadFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Enabled = false

	// Buffer a prompt event
	promptInput := `{"conversation_id": "deferred-test-1"}`
	promptReader := bytes.NewBufferString(promptInput)
	if err := ProcessEventWithEvent(promptReader, cfg, "cursor", "beforeSubmitPrompt"); err != nil {
		t.Fatalf("failed to buffer prompt: %v", err)
	}

	// Send stop event — should not error even with no server
	stopInput := `{"conversation_id": "deferred-test-1"}`
	stopReader := bytes.NewBufferString(stopInput)
	if err := ProcessEventWithEvent(stopReader, cfg, "cursor", "stop"); err != nil {
		t.Fatalf("stop event should not error: %v", err)
	}
}
```

**Step 2: Run test to verify current behavior still works**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -run TestHandleStopEvent_CreatesPayloadFile -v`
Expected: PASS (existing code path still works).

**Step 3: Rewrite `handleStopEvent`**

Replace the `handleStopEvent` function in `internal/hooks/handler.go` (lines 1005-1077) with:

```go
func handleStopEvent(sessionKey, tool string, event *models.Event, rawMap map[string]any, cfg *config.Config) error {
	cleanupStaleBuffers()

	if err := appendToBuffer(sessionKey, event, rawMap); err != nil {
		return fmt.Errorf("failed to buffer event: %w", err)
	}

	bufferedEvents, err := readAndClearBuffer(sessionKey)
	if err != nil {
		return fmt.Errorf("failed to read buffer: %w", err)
	}

	if len(bufferedEvents) == 0 {
		return nil
	}

	scan := createAggregatedScan(bufferedEvents, tool)
	if scan == nil {
		return nil
	}

	// Save scan locally if debug mode is on (sync, no network)
	if debug.Enabled {
		if err := scanner.SaveScan(scan); err != nil {
			debug.Warn("failed to save scan locally: %v", err)
		} else {
			debug.Log("Saved scan locally: %s", scan.ID)
		}
	}

	// Write payload and spawn detached child for network I/O
	payloadPath, err := writeSendPayload("send_scan", scan, scan.ID, sessionKey, "", 0)
	if err != nil {
		// Fallback: queue offline if we can't even write the payload
		debug.Warn("failed to write send payload: %v", err)
		if qErr := queue.Enqueue(scan); qErr != nil {
			debug.Warn("failed to queue scan offline: %v", qErr)
		}
		return nil
	}

	if err := spawnDetachedSend(payloadPath); err != nil {
		// Fallback: send inline (old behavior) if spawn fails
		debug.Warn("failed to spawn deferred send, falling back to inline: %v", err)
		os.Remove(payloadPath)
		return handleStopEventInline(scan, sessionKey, cfg)
	}

	return nil
}

// handleStopEventInline is the legacy inline send path, used as fallback
// when the detached process cannot be spawned.
func handleStopEventInline(scan *models.Scan, sessionKey string, cfg *config.Config) error {
	synced := false

	creds, credErr := auth.GetValidCredentials()
	if credErr != nil {
		debug.Warn("credential check failed: %v", credErr)
	}
	if creds != nil {
		if err := api.SendScanWithJWT(scan, creds.AccessToken); err != nil {
			debug.Warn("failed to sync to api.intentra.sh: %v", err)
		} else {
			debug.Log("Synced to https://api.intentra.sh")
			synced = true
		}
	}

	if !synced && cfg.Server.Enabled {
		client, err := api.NewClient(cfg)
		if err == nil {
			if err := client.SendScan(scan); err != nil {
				debug.Warn("sync failed: %v", err)
			} else {
				synced = true
			}
		}
	}

	if !synced {
		if err := queue.Enqueue(scan); err != nil {
			debug.Warn("failed to queue scan offline: %v", err)
		}
	}

	if synced && scan.ID != "" {
		saveLastScanID(sessionKey, scan.ID)
		if creds != nil {
			go queue.FlushWithJWT(creds.AccessToken)
		}
	}

	return nil
}
```

**Step 4: Run all handler tests**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -v`
Expected: All PASS.

**Step 5: Commit**

```bash
git add internal/hooks/handler.go internal/hooks/handler_deferred_test.go
git commit -m "refactor: handleStopEvent uses detached process for network I/O"
```

---

### Task 5: Rewrite `handleSessionEndEvent` to use detached send

**Files:**
- Modify: `internal/hooks/handler.go:1079-1112` (`handleSessionEndEvent`)

**Step 1: Write a test for deferred session end**

```go
// Add to internal/hooks/handler_deferred_test.go

func TestHandleSessionEndEvent_DoesNotBlock(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Enabled = false

	// First send a stop event to create a lastScanID
	promptInput := `{"session_id": "sess-end-test"}`
	promptReader := bytes.NewBufferString(promptInput)
	if err := ProcessEventWithEvent(promptReader, cfg, "claude", "PreToolUse"); err != nil {
		t.Fatalf("failed to buffer event: %v", err)
	}

	stopInput := `{"session_id": "sess-end-test"}`
	stopReader := bytes.NewBufferString(stopInput)
	if err := ProcessEventWithEvent(stopReader, cfg, "claude", "Stop"); err != nil {
		t.Fatalf("stop event should not error: %v", err)
	}

	// Now send SessionEnd — should return quickly without error
	sessionEndInput := `{"session_id": "sess-end-test", "reason": "user_exit", "duration_ms": 30000}`
	sessionEndReader := bytes.NewBufferString(sessionEndInput)
	err := ProcessEventWithEvent(sessionEndReader, cfg, "claude", "SessionEnd")
	if err != nil {
		t.Fatalf("SessionEnd should not error: %v", err)
	}
}
```

**Step 2: Run test**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -run TestHandleSessionEndEvent_DoesNotBlock -v`
Expected: PASS (current code also passes — this validates the interface contract).

**Step 3: Rewrite `handleSessionEndEvent`**

Replace the function in `internal/hooks/handler.go`:

```go
func handleSessionEndEvent(sessionKey string, rawMap map[string]any) error {
	lastScanID := getLastScanID(sessionKey)
	if lastScanID == "" {
		debug.Log("sessionEnd event but no lastScanID for session %s, ignoring", sessionKey)
		return nil
	}

	reason := ""
	durationMs := int64(0)
	if rawMap != nil {
		if r, ok := rawMap["reason"].(string); ok {
			reason = r
		}
		durationMs = extractInt64(rawMap, "duration_ms")
	}

	clearLastScanID(sessionKey)

	payloadPath, err := writeSendPayload("patch_session_end", nil, lastScanID, sessionKey, reason, durationMs)
	if err != nil {
		debug.Warn("failed to write session end payload: %v", err)
		return nil
	}

	if err := spawnDetachedSend(payloadPath); err != nil {
		debug.Warn("failed to spawn deferred session end, ignoring: %v", err)
		os.Remove(payloadPath)
	}

	return nil
}
```

**Step 4: Run all handler tests**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -v`
Expected: All PASS.

**Step 5: Commit**

```bash
git add internal/hooks/handler.go internal/hooks/handler_deferred_test.go
git commit -m "refactor: handleSessionEndEvent uses detached process for PATCH"
```

---

### Task 6: Move `saveLastScanID` into the `__send` subcommand

With the detached process handling send, the `saveLastScanID` call must move into `deferredSendScan` (in `cmd/intentra/send.go`), since the parent no longer knows if the send succeeded.

**Files:**
- Modify: `cmd/intentra/send.go` (`deferredSendScan`)
- Modify: `internal/hooks/handler.go` (export `SaveLastScanID` and `ClearLastScanID`)

**Step 1: Export the scan ID helpers**

In `internal/hooks/handler.go`, rename:
- `saveLastScanID` → `SaveLastScanID` (exported)
- `getLastScanID` → `GetLastScanID` (exported)
- `clearLastScanID` → `ClearLastScanID` (exported)
- `getLastScanPath` → `GetLastScanPath` (exported)

Update all internal call sites to use the new names.

**Step 2: Update `deferredSendScan` in send.go to call SaveLastScanID**

```go
func deferredSendScan(payload sendPayload, cfg *config.Config) error {
	scan := payload.Scan
	if scan == nil {
		return fmt.Errorf("send_scan payload missing scan")
	}

	synced := false

	creds, credErr := auth.GetValidCredentials()
	if credErr != nil {
		debug.Warn("credential check failed: %v", credErr)
	}
	if creds != nil {
		if err := api.SendScanWithJWT(scan, creds.AccessToken); err != nil {
			debug.Warn("failed to sync to api.intentra.sh: %v", err)
		} else {
			debug.Log("Synced to https://api.intentra.sh")
			synced = true
		}
	}

	if !synced && cfg.Server.Enabled {
		client, err := api.NewClient(cfg)
		if err == nil {
			if err := client.SendScan(scan); err != nil {
				debug.Warn("sync failed: %v", err)
			} else {
				synced = true
			}
		}
	}

	if !synced {
		if err := queue.Enqueue(scan); err != nil {
			debug.Warn("failed to queue scan offline: %v", err)
		}
	}

	if synced && scan.ID != "" && payload.SessionKey != "" {
		hooks.SaveLastScanID(payload.SessionKey, scan.ID)
		if creds != nil {
			queue.FlushWithJWT(creds.AccessToken)
		}
	}

	return nil
}
```

**Step 3: Run all tests**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./... -v`
Expected: All PASS.

**Step 4: Commit**

```bash
git add internal/hooks/handler.go cmd/intentra/send.go
git commit -m "refactor: export scan ID helpers, move SaveLastScanID to __send"
```

---

### Task 7: Add stale payload cleanup

Payload temp files should be cleaned up if the child process crashes before deleting them.

**Files:**
- Modify: `internal/hooks/handler.go` (`cleanupStaleBuffers`)

**Step 1: Write a test for stale payload cleanup**

```go
// Add to internal/hooks/handler_deferred_test.go

func TestCleanupStalePayloads(t *testing.T) {
	// Create a fake stale payload file
	f, err := os.CreateTemp("", "intentra_send_*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte(`{"action":"test"}`))
	f.Close()

	// Backdate it
	staleTime := time.Now().Add(-2 * time.Hour)
	os.Chtimes(f.Name(), staleTime, staleTime)

	// Remove the cleanup marker to force a cleanup run
	os.Remove(filepath.Join(os.TempDir(), cleanupMarkerFile))

	cleanupStaleBuffers()

	if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
		t.Errorf("stale payload file should have been cleaned up: %s", f.Name())
		os.Remove(f.Name()) // manual cleanup if test fails
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -run TestCleanupStalePayloads -v`
Expected: FAIL — stale payload not cleaned up yet.

**Step 3: Add payload cleanup pattern to `cleanupStaleBuffers`**

In `internal/hooks/handler.go`, in the `cleanupStaleBuffers` function, add to the `patterns` slice:

```go
	patterns := []string{
		filepath.Join(os.TempDir(), "intentra_buffer_*.jsonl"),
		filepath.Join(os.TempDir(), "intentra_lastscan_*.txt"),
		filepath.Join(os.TempDir(), "intentra_send_*.json"),
	}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./internal/hooks/ -run TestCleanupStalePayloads -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && go test ./... -race`
Expected: All PASS.

**Step 6: Commit**

```bash
git add internal/hooks/handler.go internal/hooks/handler_deferred_test.go
git commit -m "feat: clean up stale __send payload files in temp directory"
```

---

### Task 8: Final integration verification

**Files:** None (verification only)

**Step 1: Build the binary**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && make build`
Expected: Binary built to `bin/intentra`.

**Step 2: Verify `__send` subcommand works**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && ./bin/intentra __send --help`
Expected: Shows help text for the hidden command.

**Step 3: Run full test suite with race detection**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && make test`
Expected: All PASS.

**Step 4: Run linter**

Run: `cd /Users/atbabers/SecEng/intentra/intentra-cli && make lint`
Expected: No issues.

**Step 5: Simulate the Claude Code flow end-to-end**

```bash
cd /Users/atbabers/SecEng/intentra/intentra-cli

# 1. Buffer a prompt event
echo '{"session_id":"test-e2e","tool_name":"Read"}' | ./bin/intentra hook --tool claude --event PreToolUse

# 2. Fire Stop event (should return instantly, spawn child)
echo '{"session_id":"test-e2e"}' | ./bin/intentra hook --tool claude --event Stop

# 3. Fire SessionEnd (should return instantly, spawn child)
echo '{"session_id":"test-e2e","reason":"user_exit","duration_ms":5000}' | ./bin/intentra hook --tool claude --event SessionEnd
```

Expected: All three return immediately with exit code 0. No "Hook cancelled" possible since there's no blocking network I/O.

**Step 6: Verify no orphan processes**

Run: `ps aux | grep "intentra __send" | grep -v grep`
Expected: Either no results (child already finished) or a short-lived process that exits within seconds.
