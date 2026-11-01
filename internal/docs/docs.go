// Package docs serves the API's own documentation from the API process:
// the contract at /openapi.yaml and an interactive reference at /docs. They
// live with the API, not the web app, so they are reachable exactly where the
// API is: keep the API port internal and the docs stay internal too.
package docs

import (
	"net/http"

	"github.com/2SSK/go-next-boilerplate/api"
)

// Register mounts GET /openapi.yaml and GET /docs on mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", serveSpec)
	mux.HandleFunc("GET /docs", servePage)
}

func serveSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(api.Spec)
}

// page renders /openapi.yaml with Scalar (MIT), loaded from a CDN at a pinned
// version so the Go binary carries no docs UI.
const page = `<!doctype html>
<html>
<head>
  <title>API docs</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.75.0"></script>
  <script>
    Scalar.createApiReference("#app", {
      url: "/openapi.yaml",
      // Scalar reports usage by default; this page shouldn't.
      telemetry: false,
      // "Ask AI" sends the spec to Scalar's hosted service.
      agent: { disabled: true },
    });
  </script>
</body>
</html>
`

func servePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}
