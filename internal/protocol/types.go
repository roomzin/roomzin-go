package protocol

import "errors"

var (
	ErrConnClosed = errors.New("connection closed")
	ErrTimeout    = errors.New("request timed out")
)

// RawResult is what the read loop delivers to waiting calls.
type RawResult struct {
	Status string
	Fields []Field
}
