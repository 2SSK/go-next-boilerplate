package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxTitleLen matches the CHECK constraint in the todos migration and the
// maxLength in api/openapi.yaml.
const MaxTitleLen = 200

// Todo is a single item on the list. The wire type is api.Todo.
type Todo struct {
	ID        int64
	Title     string
	Completed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TodoPatch is a partial update; nil fields are left unchanged.
type TodoPatch struct {
	Title     *string
	Completed *bool
}

// NormalizeTitle trims surrounding space and enforces the length rules.
func NormalizeTitle(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return "", errors.New("title is required")
	case utf8.RuneCountInString(s) > MaxTitleLen:
		return "", errors.New("title must be at most 200 characters")
	}
	return s, nil
}
