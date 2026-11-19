// Package errs is the single error vocabulary for the CLI, API and UI (sb-errors(7)).
// Codes, HTTP statuses and exit codes come from the generated registry in codes.go.
package errs

import (
	"errors"
	"fmt"
)

// Code is a stable error code such as EPERM.
type Code string

type entry struct {
	http, exit      int
	meaning, remedy string
}

// Error carries a registry code and a one-line, user-safe message.
type Error struct {
	Code Code
	Msg  string
}

func (e *Error) Error() string { return fmt.Sprintf("%s [%s]", e.Msg, e.Code) }

// New makes an error with a registry code. Unknown codes panic: they are programmer errors.
func New(c Code, format string, args ...any) *Error {
	if _, ok := registry[c]; !ok {
		panic("errs: code not in registry: " + string(c))
	}
	return &Error{Code: c, Msg: fmt.Sprintf(format, args...)}
}

// CodeOf returns the code of err, or "" if err carries none.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// Exit is the CLI exit status for err: 0 for nil, the registry value for coded errors,
// and 1 for anything uncoded (an uncoded error is a failure, never a success).
func Exit(err error) int {
	if err == nil {
		return 0
	}
	if r, ok := registry[CodeOf(err)]; ok {
		return r.exit
	}
	return 1
}

// HTTP is the HTTP status for a code, or 0 if the code has none (CLI-only codes).
func HTTP(c Code) int { return registry[c].http }

// Remedy is the one-line remedy from the manual.
func Remedy(c Code) string { return registry[c].remedy }

// Codes lists every registry code.
func Codes() []Code {
	out := make([]Code, 0, len(registry))
	for c := range registry {
		out = append(out, c)
	}
	return out
}
