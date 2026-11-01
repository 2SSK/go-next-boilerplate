package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("production logs JSON at info", func(t *testing.T) {
		var buf bytes.Buffer
		log := New("production", &buf)
		log.Debug("hidden")
		log.Info("hello", "k", "v")

		var line map[string]any
		if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
			t.Fatalf("want one JSON line, got %q: %v", buf.String(), err)
		}
		if line["msg"] != "hello" || line["k"] != "v" {
			t.Errorf("line = %v", line)
		}
	})

	t.Run("development logs text at debug", func(t *testing.T) {
		var buf bytes.Buffer
		New("development", &buf).Debug("visible")
		if !strings.Contains(buf.String(), "level=DEBUG msg=visible") {
			t.Errorf("got %q", buf.String())
		}
	})
}
