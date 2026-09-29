package studio

import "github.com/mxschmitt/playwright-go"

// dismissCoachMarks dismisses YouTube feature-discovery banners only.
// Do not click generic "Close" or press Escape — the upload dialog uses those to cancel.
func dismissCoachMarks(page playwright.Page) {
	ms := 2000.0
	selectors := []string{
		`ytcp-feature-discovery-banner ytcp-button:has-text("Got it")`,
		`ytcp-feature-discovery-dialog ytcp-button:has-text("Got it")`,
		`ytce-guide-with-learn-more ytcp-button:has-text("Got it")`,
	}
	for _, sel := range selectors {
		loc := page.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		_ = loc.Click(playwright.LocatorClickOptions{Timeout: &ms})
	}
}

func uploadDialog(page playwright.Page) playwright.Locator {
	return page.Locator("ytcp-uploads-dialog").First()
}
