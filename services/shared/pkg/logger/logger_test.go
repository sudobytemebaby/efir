package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}

	logger := New(Options{
		Level:  LevelInfo,
		Output: buf,
	})

	logger.Info("test message", "key", "value")

	if buf.Len() == 0 {
		t.Error("expected JSON output, got empty string")
	}

	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Error("expected log to contain message")
	}
}

func TestWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Options{
		Level:  LevelInfo,
		Output: buf,
	})

	ctx := WithContext(context.Background(), logger)
	retrieved := FromContext(ctx)

	if retrieved != logger {
		t.Error("expected retrieved logger to be the same as original")
	}
}

func TestFromContextDefault(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)

	if logger == nil {
		t.Error("expected default logger, got nil")
	}
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		input    string
		expected slog.Level
		wantErr  bool
	}{
		{"debug", LevelDebug, false},
		{"info", LevelInfo, false},
		{"warn", LevelWarn, false},
		{"error", LevelError, false},
		{"DEBUG", LevelDebug, false}, // case-insensitive
		{"INFO", LevelInfo, false},
		{"", LevelInfo, true},
		{"verbose", LevelInfo, true},
		{"trace", LevelInfo, true},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			level, err := ParseLevel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tc.input, err)
				}
				if level != tc.expected {
					t.Errorf("expected %v, got %v", tc.expected, level)
				}
			}
		})
	}
}

func TestLevels(t *testing.T) {
	buf := &bytes.Buffer{}

	logger := New(Options{
		Level:  LevelWarn,
		Output: buf,
	})

	logger.Debug("debug should not appear")
	logger.Info("info should not appear")
	logger.Warn("warn should appear")
	logger.Error("error should appear")

	if contains(buf.Bytes(), "debug should not appear") {
		t.Error("debug level should not be logged")
	}
	if contains(buf.Bytes(), "info should not appear") {
		t.Error("info level should not be logged")
	}
	if !contains(buf.Bytes(), "warn should appear") {
		t.Error("warn level should be logged")
	}
	if !contains(buf.Bytes(), "error should appear") {
		t.Error("error level should be logged")
	}
	_ = buf.String()
}

func contains(b []byte, s string) bool {
	return len(b) > 0 && bytes.Contains(b, []byte(s))
}
