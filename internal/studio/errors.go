package studio

import (
	"errors"
	"strings"
)

func containsTimeout(msg string) bool {
	return strings.Contains(msg, "timed out")
}

var (
	ErrNeedsAuth      = errors.New("youtube session requires authentication")
	ErrStudioState    = errors.New("unexpected youtube studio state")
	ErrUploadTimeout  = errors.New("upload timed out")
	ErrScheduleNotReady = errors.New("scheduled publishing is not implemented yet")
)

type StepError struct {
	Step string
	Err  error
}

func (e *StepError) Error() string {
	return e.Step + ": " + e.Err.Error()
}

func (e *StepError) Unwrap() error {
	return e.Err
}
