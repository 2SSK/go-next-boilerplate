// Package models holds the domain types and their rules: what a value may be,
// independent of how it is stored (internal/store) or sent over HTTP (the api
// package, generated from the contract). It imports neither.
package models

import "errors"

// ErrNotFound is returned by stores when the requested record does not exist.
// Handlers map it to a 404.
var ErrNotFound = errors.New("not found")
