package types

import (
	"errors"
	"fmt"
	"strings"
)

// ---------- public error types ----------

type ErrorKind uint8

const (
	KindClient   ErrorKind = iota // SDK misuse / auth / config
	KindRequest                   // validation, not-found, overflow …
	KindInternal                  // bug, parse failure, protocol mismatch
	KindRetry                     // 429, 503, 308, role change …
)

// Error satisfies error and gives access to the kind.
type Error struct {
	Kind ErrorKind
	Code string // original code, e.g. "AUTH_ERROR"
	Msg  string // human message without prefix
}

func (e *Error) Error() string { return fmt.Sprintf("%s:%s", e.Code, e.Msg) }

// ---------- helpers for users ----------

func IsClient(err error) bool   { return isKind(err, KindClient) }
func IsRequest(err error) bool  { return isKind(err, KindRequest) }
func IsInternal(err error) bool { return isKind(err, KindInternal) }
func IsCluster(err error) bool  { return isKind(err, KindRetry) }

// errors.Is support
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind && t.Code == e.Code
}

// ---------- internal: wrap anything that comes from the wire ----------
// RzError is the single internal entry-point.
// It accepts either:
//   - a plain Go error produced inside the SDK
//   - a string that came from the server
//
// and always returns *Error.
// RzError turns anything into *Error.
// If the caller already knows the bucket, pass it; otherwise we guess.
func RzError(in any, kind ...ErrorKind) *Error {
	if in == nil {
		return nil
	}

	// 1. already wrapped?
	var e *Error
	if errors.As(anyToError(in), &e) {
		return e
	}

	// 2. explicit bucket?
	if len(kind) > 0 {
		return &Error{
			Kind: kind[0],
			Code: codeOf(kind[0]),
			Msg:  extractMsg(in),
		}
	}

	// 3. server string?
	if s, ok := in.(string); ok {
		code, msg, _ := strings.Cut(s, ":")
		return classify(code, msg)
	}

	// 4. default → internal
	return &Error{
		Kind: KindInternal,
		Code: "INTERNAL_ERROR",
		Msg:  extractMsg(in),
	}
}

// ---------- helper ----------
func extractMsg(v any) string {
	if err, ok := v.(error); ok {
		return err.Error() // <-- explicit use of .Error()
	}
	return fmt.Sprint(v)
}

func codeOf(k ErrorKind) string {
	switch k {
	case KindClient:
		return "CLIENT_ERROR"
	case KindRequest:
		return "REQUEST_ERROR"
	case KindRetry:
		return "RETRY_ERROR"
	default:
		return "INTERNAL_ERROR"
	}
}

// anyToError is a tiny helper that turns any into error.
func anyToError(v any) error {
	switch t := v.(type) {
	case error:
		return t
	case string:
		return errors.New(t)
	default:
		return fmt.Errorf("%v", v)
	}
}

func classify(code, msg string) *Error {
	switch code {
	case "AUTH_ERROR":
		return &Error{Kind: KindClient, Code: code, Msg: msg}
	case "VALIDATION_ERROR", "NOT_FOUND", "OVERFLOW", "UNDERFLOW", "FORBIDDEN":
		return &Error{Kind: KindRequest, Code: code, Msg: msg}
	case "503", "429", "308", "405":
		return &Error{Kind: KindRetry, Code: code, Msg: msg}
	default:
		// "PARSE_ERROR" , "RESPONSE_ERROR"
		return &Error{Kind: KindInternal, Code: code, Msg: msg}
	}
}

func isKind(err error, want ErrorKind) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == want
}
