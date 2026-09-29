package studio

import (
	"context"
	"fmt"
	"time"
)

func runStep(parent context.Context, step string, timeout time.Duration, fn func(context.Context) error) error {
	if timeout <= 0 {
		return fn(parent)
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- fn(ctx) }()

	select {
	case err := <-done:
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("step %s timed out after %s", step, timeout)
		}
		return err
	case <-ctx.Done():
		return fmt.Errorf("step %s timed out after %s", step, timeout)
	}
}
