package browser

import (
	"fmt"
	"strings"

	"github.com/mxschmitt/playwright-go"
)

// LaunchSettings controls Chromium/Chrome startup for persistent profiles.
type LaunchSettings struct {
	UserDataDir    string
	Headless       bool
	TimeoutMS      float64
	Channel        string
	ExecutablePath string
	Stealth        bool
}

func applyPersistentOptions(s LaunchSettings) playwright.BrowserTypeLaunchPersistentContextOptions {
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: &s.Headless,
	}
	if s.Channel != "" {
		opts.Channel = &s.Channel
	}
	if s.ExecutablePath != "" {
		opts.ExecutablePath = &s.ExecutablePath
	}
	if s.Stealth {
		opts.IgnoreDefaultArgs = []string{"--enable-automation"}
		opts.Args = []string{
			"--disable-blink-features=AutomationControlled",
		}
	}
	return opts
}

const stealthInitScript = `
(() => {
  Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
})();
`

func launchPersistent(pw *playwright.Playwright, s LaunchSettings) (playwright.BrowserContext, error) {
	opts := applyPersistentOptions(s)
	if s.TimeoutMS > 0 {
		opts.Timeout = &s.TimeoutMS
	}

	bctx, err := pw.Chromium.LaunchPersistentContext(s.UserDataDir, opts)
	if err != nil {
		return nil, err
	}
	if s.Stealth {
		content := stealthInitScript
		if err := bctx.AddInitScript(playwright.Script{Content: &content}); err != nil {
			bctx.Close()
			return nil, fmt.Errorf("add init script: %w", err)
		}
	}
	return bctx, nil
}

func launchWithFallback(pw *playwright.Playwright, s LaunchSettings) (playwright.BrowserContext, error) {
	bctx, err := launchPersistent(pw, s)
	if err == nil {
		return bctx, nil
	}
	if s.Channel == "" {
		return nil, err
	}
	// Retry without branded channel (e.g. Chrome not installed in CI/Docker).
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "chrome") || strings.Contains(msg, "channel") || strings.Contains(msg, "executable") {
		fallback := s
		fallback.Channel = ""
		return launchPersistent(pw, fallback)
	}
	return nil, err
}
