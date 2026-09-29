package studio

import (
	"fmt"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

func (u *Uploader) verifyPublish(page playwright.Page) (string, error) {
	deadline := time.Now().Add(u.StepVerify)
	for time.Now().Before(deadline) {
		if u.isPublishComplete(page) {
			url := extractYouTubeVideoURL(page)
			dismissPostPublishDialogs(page)
			return url, nil
		}
		dismissPostPublishDialogs(page)
		page.WaitForTimeout(400)
	}
	return "", fmt.Errorf("publish verification timed out after %s", u.StepVerify)
}

func (u *Uploader) isPublishComplete(page playwright.Page) bool {
	if isPublishSignalVisible(page) {
		return true
	}
	if !uploadDialogOpen(page) {
		if extractYouTubeVideoURL(page) != "" {
			return true
		}
	}
	cur := page.URL()
	if strings.Contains(cur, "/video/") && strings.Contains(cur, "studio.youtube.com") {
		return true
	}
	return false
}

func isPublishSignalVisible(page playwright.Page) bool {
	signals := []string{
		`text=/Video published/i`,
		`text=/Processing will begin/i`,
		`text=/still being processed/i`,
		`text=/processing to SD/i`,
		`text=/being processed to SD/i`,
		`ytcp-video-share-dialog`,
		`ytcp-post-publish-dialog`,
		`a[href*="youtu.be/"]`,
		`#share-url`,
		`text=/Video link/i`,
	}
	for _, sel := range signals {
		if visible(page, sel) {
			return true
		}
	}
	return false
}

func uploadDialogOpen(page playwright.Page) bool {
	d := uploadDialog(page)
	n, err := d.Count()
	if err != nil || n == 0 {
		return false
	}
	ok, err := d.IsVisible()
	return err == nil && ok
}

func extractYouTubeVideoURL(page playwright.Page) string {
	selectors := []string{
		`a[href*="youtu.be/"]`,
		`a[href*="youtube.com/watch"]`,
		`#share-url`,
		`input[value*="youtu.be"]`,
		`ytcp-video-share-dialog a`,
	}
	for _, sel := range selectors {
		loc := page.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		if href, err := loc.GetAttribute("href"); err == nil && strings.Contains(href, "youtu") {
			return href
		}
		if val, err := loc.GetAttribute("value"); err == nil && strings.Contains(val, "youtu") {
			return val
		}
		text, err := loc.InnerText()
		if err == nil && strings.Contains(text, "youtu") {
			return strings.TrimSpace(text)
		}
	}
	cur := page.URL()
	if strings.Contains(cur, "studio.youtube.com") && strings.Contains(cur, "/video/") {
		return cur
	}
	return ""
}

// dismissPostPublishDialogs closes success / processing modals after publish (not the upload wizard).
func dismissPostPublishDialogs(page playwright.Page) {
	ms := 2500.0
	if uploadDialogOpen(page) {
		return
	}
	selectors := []string{
		`ytcp-video-share-dialog ytcp-button[aria-label="Close"]`,
		`ytcp-post-publish-dialog ytcp-button[aria-label="Close"]`,
		`tp-yt-paper-dialog:not(ytcp-uploads-dialog) ytcp-button[aria-label="Close"]`,
		`tp-yt-paper-dialog:not(ytcp-uploads-dialog) #close-button`,
	}
	for _, sel := range selectors {
		loc := page.Locator(sel).First()
		n, err := loc.Count()
		if err != nil || n == 0 {
			continue
		}
		ok, err := loc.IsVisible()
		if err != nil || !ok {
			continue
		}
		_ = loc.Click(playwright.LocatorClickOptions{Timeout: &ms})
		return
	}
	// SD / processing info dialog often has a plain "Close" button.
	if visible(page, `text=/still being processed/i`) || visible(page, `text=/processing to SD/i`) {
		btn := page.Locator(`tp-yt-paper-dialog:not(ytcp-uploads-dialog)`).GetByRole("button", playwright.LocatorGetByRoleOptions{Name: "Close"})
		if n, _ := btn.Count(); n > 0 {
			_ = btn.First().Click(playwright.LocatorClickOptions{Timeout: &ms})
		}
	}
}
