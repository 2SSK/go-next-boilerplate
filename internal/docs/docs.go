// Package docs serves the API's own documentation from the API process:
// the contract at /openapi.yaml and an interactive reference at /docs. They
// live with the API, not the web app, so they are reachable exactly where the
// API is; DOCS_ENABLED=false turns them off entirely.
package docs

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/2SSK/go-next-boilerplate/api"
)

// Path is where the interactive reference is served.
const Path = "/docs"

// Register mounts GET /openapi.yaml and GET /docs on mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", serveSpec)
	mux.HandleFunc("GET "+Path, servePage)
}

func serveSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(api.Spec)
}

// scalarURL is Scalar's API reference (MIT), pinned so the page can't change
// underneath us and the Go binary carries no docs UI.
const scalarURL = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.75.0"

// initScript starts the reference. Its hash, not 'unsafe-inline', is what the
// page's Content-Security-Policy allows, so no other inline script can run.
const initScript = `
    Scalar.createApiReference("#app", {
      url: "/openapi.yaml",
      // Scalar reports usage by default; this page shouldn't.
      telemetry: false,
      // "Ask AI" sends the spec to Scalar's hosted service.
      agent: { disabled: true },
    });
  `

var page = `<!doctype html>
<html>
<head>
  <title>API docs</title>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <div id="app"></div>
  <script src="` + scalarURL + `"></script>
  <script>` + initScript + `</script>
</body>
</html>
`

// pagePolicy allows exactly what the reference needs: Scalar's script from the
// CDN, the init script by hash, its injected styles and fonts, and fetching
// the spec from this origin.
var pagePolicy = "default-src 'none'; " +
	"script-src " + scalarURL + " 'sha256-" + sha256Base64(initScript) + "'; " +
	"style-src 'unsafe-inline'; " +
	"font-src https://fonts.scalar.com data:; " +
	"img-src 'self' data: https:; " +
	"connect-src 'self'; " +
	"base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func servePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", pagePolicy)
	_, _ = w.Write([]byte(page))
}

func sha256Base64(s string) string {
	sum := sha256.Sum256([]byte(s))
	return base64.StdEncoding.EncodeToString(sum[:])
}
