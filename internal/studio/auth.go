package studio

import (
	"strings"
	"time"

	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/mxschmitt/playwright-go"
)

func classifyURL(url string) domain.SessionStatus {
	u := strings.ToLower(url)
	switch {
	case strings.Contains(u, "accounts.google.com"):
		if strings.Contains(u, "challenge") ||
			strings.Contains(u, "signin/v2/challenge") ||
			strings.Contains(u, "interstitial") {
			return domain.SessionNeedsUserAction
		}
		return domain.SessionNeedsReauth
	case strings.Contains(u, "servicelogin"):
		return domain.SessionNeedsReauth
	case strings.Contains(u, "accounts.google.com/signin"):
		return domain.SessionNeedsReauth
	default:
		return ""
	}
}

type DetectResult struct {
	Status      domain.SessionStatus
	AccountHint string
}

func DetectOnPage(page playwright.Page, studioURL string, navTimeout time.Duration) (DetectResult, error) {
	ms := float64(navTimeout.Milliseconds())
	_, err := page.Goto(studioURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   &ms,
	})
	if err != nil {
		return DetectResult{Status: domain.SessionCheckFailed}, err
	}
	return waitForSessionState(page, navTimeout), nil
}

func waitForSessionState(page playwright.Page, budget time.Duration) DetectResult {
	deadline := time.Now().Add(budget)
	var last DetectResult

	for time.Now().Before(deadline) {
		last = DetectCurrent(page)
		switch last.Status {
		case domain.SessionActive, domain.SessionNeedsUserAction:
			return last
		case domain.SessionNeedsReauth:
			if definiteSignOut(page) {
				return last
			}
		}
		page.WaitForTimeout(500)
	}
	if last.Status == "" || (!definiteSignOut(page) && onStudioHost(page.URL())) {
		final := DetectCurrent(page)
		if final.Status == domain.SessionActive {
			return final
		}
	}
	return last
}

func definiteSignOut(page playwright.Page) bool {
	if classifyURL(page.URL()) != "" {
		return true
	}
	return visible(page, `text=/Sign in/i`)
}

func onStudioHost(rawURL string) bool {
	u := strings.ToLower(rawURL)
	return strings.Contains(u, "studio.youtube.com")
}

func DetectCurrent(page playwright.Page) DetectResult {
	if st := classifyURL(page.URL()); st != "" {
		return DetectResult{Status: st}
	}

	if visible(page, `text=/Sign in/i`) {
		return DetectResult{Status: domain.SessionNeedsReauth}
	}

	if studioSignedIn(page) {
		return DetectResult{
			Status:      domain.SessionActive,
			AccountHint: accountHint(page),
		}
	}

	return DetectResult{Status: domain.SessionNeedsReauth}
}

func studioSignedIn(page playwright.Page) bool {
	selectors := []string{
		`ytcp-app`,
		`#create-icon`,
		`[aria-label="Create"]`,
		`ytcp-navigation-drawer`,
		`#upload-icon`,
		`tp-yt-paper-icon-button#avatar-btn`,
	}
	for _, sel := range selectors {
		if visible(page, sel) {
			return true
		}
	}
	return false
}

func visible(page playwright.Page, selector string) bool {
	loc := page.Locator(selector).First()
	n, err := loc.Count()
	if err != nil || n == 0 {
		return false
	}
	ok, err := loc.IsVisible()
	return err == nil && ok
}

func accountHint(page playwright.Page) string {
	selectors := []string{
		`#channel-title`,
		`ytcp-channel-title`,
		`#account-name`,
		`[id*="channel-name"]`,
	}
	for _, sel := range selectors {
		if !visible(page, sel) {
			continue
		}
		loc := page.Locator(sel).First()
		text, err := loc.InnerText()
		if err == nil && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}
