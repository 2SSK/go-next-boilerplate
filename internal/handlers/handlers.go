// Package handlers implements the API: every operation in api/openapi.yaml, as
// methods on one handler type per area. The generated code decodes requests
// and encodes responses; handlers apply the rules in internal/models and talk
// to storage through small interfaces they declare, so tests use fakes.
//
// Return a typed response for every expected outcome (404, 422, ...), built
// with api.NewBadRequest, api.NewNotFound, and api.NewValidationFailed. Return
// a Go error only for the unexpected: the router logs it and answers a generic
// 500, so database details never reach the client.
package handlers

import "github.com/2SSK/go-next-boilerplate/api"

// Handlers is the whole api.StrictServerInterface, assembled from one handler
// per area. A feature adds its operations by embedding its handler here and
// constructing it in router.Routes.
type Handlers struct {
	*SystemHandler
	*TodoHandler // feature:todo
}

// The compiler checks that every operation in the spec is implemented.
var _ api.StrictServerInterface = Handlers{}
