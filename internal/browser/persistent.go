package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/mxschmitt/playwright-go"
)

type PersistentOptions struct {
	UserDataDir    string
	Headless       bool
	Timeout        time.Duration
	Channel        string
	ExecutablePath string
	Stealth        bool
}

func WithPersistentContext(ctx context.Context, opts PersistentOptions, fn func(playwright.BrowserContext) error) error {
	if opts.UserDataDir == "" {
		return fmt.Errorf("userDataDir is required")
	}
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("playwright run: %w", err)
	}
	defer pw.Stop()

	var timeoutMS float64
	if opts.Timeout > 0 {
		timeoutMS = float64(opts.Timeout.Milliseconds())
	}

	settings := LaunchSettings{
		UserDataDir:    opts.UserDataDir,
		Headless:       opts.Headless,
		TimeoutMS:      timeoutMS,
		Channel:        opts.Channel,
		ExecutablePath: opts.ExecutablePath,
		Stealth:        opts.Stealth,
	}

	bctx, err := launchWithFallback(pw, settings)
	if err != nil {
		return fmt.Errorf("launch persistent context: %w", err)
	}
	defer bctx.Close()

	if err := fn(bctx); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func FirstPage(bctx playwright.BrowserContext) (playwright.Page, error) {
	pages := bctx.Pages()
	if len(pages) > 0 {
		return pages[0], nil
	}
	return bctx.NewPage()
}
