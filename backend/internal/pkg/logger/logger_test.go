package logger

import (
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestInit_DualOutput(t *testing.T) {
	// Use os.MkdirTemp instead of t.TempDir to avoid cleanup failures
	// when lumberjack holds file handles on Windows.
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	logPath := filepath.Join(tmpDir, "logs", "sub2api.log")

	origStdout := os.Stdout
	origStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = stdoutR.Close()
		_ = stderrR.Close()
		_ = stdoutW.Close()
		_ = stderrW.Close()
	})

	err = Init(InitOptions{
		Level:       "debug",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Output: OutputOptions{
			ToStdout: true,
			ToFile:   true,
			FilePath: logPath,
		},
		Rotation: RotationOptions{
			MaxSizeMB:  10,
			MaxBackups: 2,
			MaxAgeDays: 1,
		},
		Sampling: SamplingOptions{Enabled: false},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	L().Info("dual-output-info")
	L().Warn("dual-output-warn")

	// Skip Sync() — on Windows, fsync on pipes deadlocks (FlushFileBuffers).
	// The log data is already in the pipe buffer; closing writers is sufficient.

	_ = stdoutW.Close()
	_ = stderrW.Close()
	stdoutBytes, _ := io.ReadAll(stdoutR)
	stderrBytes, _ := io.ReadAll(stderrR)
	stdoutText := string(stdoutBytes)
	stderrText := string(stderrBytes)

	if !strings.Contains(stdoutText, "dual-output-info") {
		t.Fatalf("stdout missing info log: %s", stdoutText)
	}
	if !strings.Contains(stderrText, "dual-output-warn") {
		t.Fatalf("stderr missing warn log: %s", stderrText)
	}

	fileBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	fileText := string(fileBytes)
	if !strings.Contains(fileText, "dual-output-info") || !strings.Contains(fileText, "dual-output-warn") {
		t.Fatalf("file missing logs: %s", fileText)
	}
}

func TestInit_FileOutputFailureDowngrade(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	_, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = stdoutW.Close()
		_ = stderrR.Close()
		_ = stderrW.Close()
	})

	err = Init(InitOptions{
		Level:  "info",
		Format: "json",
		Output: OutputOptions{
			ToStdout: true,
			ToFile:   true,
			FilePath: filepath.Join(os.DevNull, "logs", "sub2api.log"),
		},
		Rotation: RotationOptions{
			MaxSizeMB:  10,
			MaxBackups: 1,
			MaxAgeDays: 1,
		},
	})
	if err != nil {
		t.Fatalf("Init() should downgrade instead of failing, got: %v", err)
	}

	_ = stderrW.Close()
	stderrBytes, _ := io.ReadAll(stderrR)
	if !strings.Contains(string(stderrBytes), "日志文件输出初始化失败") {
		t.Fatalf("stderr should contain fallback warning, got: %s", string(stderrBytes))
	}
}

func TestInit_CallerShouldPointToCallsite(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	_, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		_ = stderrW.Close()
	})

	if err := Init(InitOptions{
		Level:       "info",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Caller:      true,
		Output: OutputOptions{
			ToStdout: true,
			ToFile:   false,
		},
		Sampling: SamplingOptions{Enabled: false},
	}); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	L().Info("caller-check")
	// Skip Sync() — on Windows, fsync on pipes deadlocks (FlushFileBuffers).
	os.Stdout = origStdout
	os.Stderr = origStderr
	_ = stdoutW.Close()
	logBytes, _ := io.ReadAll(stdoutR)

	var line string
	for _, item := range strings.Split(string(logBytes), "\n") {
		if strings.Contains(item, "caller-check") {
			line = item
			break
		}
	}
	if line == "" {
		t.Fatalf("log output missing caller-check: %s", string(logBytes))
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("parse log json failed: %v, line=%s", err, line)
	}
	caller, _ := payload["caller"].(string)
	if !strings.Contains(caller, "logger_test.go:") {
		t.Fatalf("caller should point to this test file, got: %s", caller)
	}
}

type captureSink struct {
	mu     sync.Mutex
	events []*LogEvent
}

func (s *captureSink) WriteLogEvent(event *LogEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyEvent := *event
	copyEvent.Fields = make(map[string]any, len(event.Fields))
	for key, value := range event.Fields {
		copyEvent.Fields[key] = value
	}
	s.events = append(s.events, &copyEvent)
}

func (s *captureSink) JSON(t *testing.T) string {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, err := json.Marshal(s.events)
	if err != nil {
		t.Fatalf("marshal captured log events: %v", err)
	}
	return string(encoded)
}

func TestSensitiveLeakageCanaryIsRedactedAcrossLoggerOutputs(t *testing.T) {
	const canary = "sub2api-canary-secret-logger"
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "sub2api.log")
	sink := &captureSink{}
	SetSink(sink)
	t.Cleanup(func() { SetSink(nil) })

	err := Init(InitOptions{
		Level:       "debug",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Output: OutputOptions{
			ToStdout: false,
			ToFile:   true,
			FilePath: logPath,
		},
		Rotation: RotationOptions{MaxSizeMB: 10, MaxBackups: 1, MaxAgeDays: 1},
		Sampling: SamplingOptions{Enabled: false},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	L().Error(
		"upstream failed Authorization: Bearer "+canary,
		zap.String("refresh_token", canary),
		zap.Any("payload", map[string]any{"proxyPassword": canary}),
		zap.Any("typed_payload", struct {
			AccessToken string `json:"accessToken"`
		}{AccessToken: canary}),
	)
	log.Printf("Cookie: session=%s", canary)
	slog.Warn("proxy check", "proxy_url", "http://worker:"+canary+"@127.0.0.1:8080")
	WriteSinkEvent("warn", "security", "raw_prompt="+canary, map[string]any{"private_key": canary})
	Sync()

	fileBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	fileOutput := string(fileBytes)
	sinkOutput := sink.JSON(t)
	for name, output := range map[string]string{"file": fileOutput, "sink": sinkOutput} {
		if strings.Contains(output, canary) {
			t.Fatalf("%s 日志泄漏完整 canary: %s", name, output)
		}
		if !strings.Contains(output, "***") {
			t.Fatalf("%s 日志缺少脱敏标记: %s", name, output)
		}
	}
}
