// Package api is the HTTP contract: openapi.yaml, and the Go server interface
// and types generated from it (api.gen.go). Handlers implement
// StrictServerInterface (internal/handlers); internal/router mounts them.
package api

import _ "embed"

//go:generate go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml

// Spec is openapi.yaml, served at /openapi.yaml for the docs page and clients.
//
//go:embed openapi.yaml
var Spec []byte
