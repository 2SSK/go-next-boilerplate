package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestTraceIDComesFromContext(t *testing.T) {
	var buf bytes.Buffer
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x0a, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	prod(&buf, slog.LevelInfo).InfoContext(ctx, "traced")
	prod(&buf, slog.LevelInfo).Info("untraced")

	lines := jsonLines(t, &buf)
	if lines[0]["trace_id"] != sc.TraceID().String() || lines[0]["span_id"] != sc.SpanID().String() {
		t.Errorf("trace_id/span_id = %v/%v, want %s/%s", lines[0]["trace_id"], lines[0]["span_id"], sc.TraceID(), sc.SpanID())
	}
	if _, ok := lines[1]["trace_id"]; ok {
		t.Error("a line without a span got a trace_id")
	}
}

func TestExport(t *testing.T) {
	newLogger := func(stdout, export *bytes.Buffer) *slog.Logger {
		// The export destination accepts everything; New must apply Level.
		exp := slog.NewJSONHandler(export, &slog.HandlerOptions{Level: slog.LevelDebug})
		return New(Options{Env: "production", Level: slog.LevelInfo, Service: "svc", Export: exp}, stdout)
	}

	t.Run("receives every line stdout gets, with secrets already redacted", func(t *testing.T) {
		var stdout, export bytes.Buffer
		log := newLogger(&stdout, &export).With("api_key", "k1")
		log.Info("login", "password", "hunter2", slog.Group("db", slog.String("db_password", "pw")))

		for name, out := range map[string]string{"stdout": stdout.String(), "export": export.String()} {
			if !strings.Contains(out, `"msg":"login"`) {
				t.Errorf("%s missed the line: %s", name, out)
			}
			for _, secret := range []string{"hunter2", `"pw"`, `"k1"`} {
				if strings.Contains(out, secret) {
					t.Errorf("%s leaked %s: %s", name, secret, out)
				}
			}
		}
	})

	t.Run("respects the level", func(t *testing.T) {
		var stdout, export bytes.Buffer
		newLogger(&stdout, &export).Debug("noise")
		if export.Len() != 0 || stdout.Len() != 0 {
			t.Errorf("debug line written below info: stdout %q, export %q", stdout.String(), export.String())
		}
	})
}
