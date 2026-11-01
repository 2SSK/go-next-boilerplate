package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func jsonLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("not a JSON line: %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func prod(buf *bytes.Buffer, level slog.Level) *slog.Logger {
	return New(Options{Env: "production", Level: level, Service: "svc", Version: "v1.2.3"}, buf)
}

func TestProductionLinesCarryServiceVersionEnv(t *testing.T) {
	var buf bytes.Buffer
	prod(&buf, slog.LevelInfo).Info("hello", "k", "v")

	lines := jsonLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
	for key, want := range map[string]any{"msg": "hello", "level": "INFO", "service": "svc", "version": "v1.2.3", "env": "production", "k": "v"} {
		if lines[0][key] != want {
			t.Errorf("%s = %v, want %v", key, lines[0][key], want)
		}
	}
}

func TestLevelFiltersOutput(t *testing.T) {
	var buf bytes.Buffer
	log := prod(&buf, slog.LevelWarn)
	log.Info("dropped")
	log.Warn("kept")

	lines := jsonLines(t, &buf)
	if len(lines) != 1 || lines[0]["msg"] != "kept" {
		t.Errorf("lines = %v, want only the warning", lines)
	}
}

func TestRequestIDComesFromContext(t *testing.T) {
	var buf bytes.Buffer
	log := prod(&buf, slog.LevelInfo)

	ctx := WithRequestID(context.Background(), "req-42")
	log.InfoContext(ctx, "inside a request")
	log.Info("outside a request")

	lines := jsonLines(t, &buf)
	if lines[0]["request_id"] != "req-42" {
		t.Errorf("request_id = %v, want req-42", lines[0]["request_id"])
	}
	if _, ok := lines[1]["request_id"]; ok {
		t.Errorf("a line without a request context got request_id %v", lines[1]["request_id"])
	}
	if RequestID(ctx) != "req-42" || RequestID(context.Background()) != "" {
		t.Error("RequestID does not round-trip")
	}
}

func TestSecretsAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	prod(&buf, slog.LevelInfo).Info("login",
		"user", "ana",
		"password", "hunter2",
		"Authorization", "Bearer abc",
		slog.Group("db", slog.String("db_password", "pw"), slog.String("host", "db")),
		"api_key", "k",
		"session_token", "t",
	)

	out := buf.String()
	for _, secret := range []string{"hunter2", "Bearer abc", `"pw"`, `"k"`, `"t"`} {
		if strings.Contains(out, secret) {
			t.Errorf("secret %s leaked: %s", secret, out)
		}
	}
	for _, kept := range []string{`"user":"ana"`, `"host":"db"`} {
		if !strings.Contains(out, kept) {
			t.Errorf("ordinary field %s was lost: %s", kept, out)
		}
	}
}

func TestErrorsCarrySourceOtherLinesDont(t *testing.T) {
	var buf bytes.Buffer
	log := prod(&buf, slog.LevelInfo)
	log.Info("fine")
	log.Error("failed", "error", errors.New("boom"))

	lines := jsonLines(t, &buf)
	if _, ok := lines[0]["source"]; ok {
		t.Error("an info line carries source; only errors should")
	}
	src, _ := lines[1]["source"].(string)
	if !strings.Contains(src, "logger_test.go:") {
		t.Errorf("error source = %q, want this file and line", src)
	}
}

func TestDevelopmentIsReadableText(t *testing.T) {
	var buf bytes.Buffer
	New(Options{Env: "development", Level: slog.LevelDebug, Service: "svc"}, &buf).Debug("visible")
	if out := buf.String(); !strings.Contains(out, "level=DEBUG") || !strings.Contains(out, "msg=visible") {
		t.Errorf("got %q, want a text line at debug", out)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in      string
		env     string
		want    slog.Level
		wantErr bool
	}{
		{"", "production", slog.LevelInfo, false},
		{"", "development", slog.LevelDebug, false},
		{"warn", "production", slog.LevelWarn, false},
		{"ERROR", "development", slog.LevelError, false},
		{"loud", "production", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in+"/"+tt.env, func(t *testing.T) {
			got, err := ParseLevel(tt.in, tt.env)
			if (err != nil) != tt.wantErr || (!tt.wantErr && got != tt.want) {
				t.Errorf("ParseLevel(%q, %q) = %v, %v", tt.in, tt.env, got, err)
			}
		})
	}
}
