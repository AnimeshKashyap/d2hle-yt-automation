package studio

import (
	"fmt"
	"strings"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// uploadProgressFromLabel interprets YouTube Studio's ytcp-video-upload-progress label text.
// Do not treat "100%" as complete — Studio can show 100% before the file transfer finishes.
// After the file transfer, Studio may show only post-upload processing text (no "Upload complete").
func uploadProgressFromLabel(label string) (complete, inProgress bool) {
	label = strings.TrimSpace(label)
	if label == "" {
		return false, false
	}
	lower := strings.ToLower(label)
	if strings.Contains(lower, "uploading") || strings.Contains(label, "%") {
		return false, true
	}
	if uploadTransferCompleteLabel(lower) {
		return true, false
	}
	return false, false
}

func uploadTransferCompleteLabel(lower string) bool {
	if strings.Contains(lower, "upload complete") {
		return true
	}
	if strings.Contains(lower, "processing will begin shortly") {
		return true
	}
	if strings.Contains(lower, "processing up to hd") {
		return true
	}
	return false
}

func readUploadProgressLabels(page playwright.Page) []string {
	scopes := []playwright.Locator{
		uploadDialog(page).Locator(`ytcp-video-upload-progress span.progress-label`),
		page.Locator(`ytcp-video-upload-progress span.progress-label`),
	}
	var labels []string
	seen := make(map[string]struct{})
	for _, scope := range scopes {
		n, err := scope.Count()
		if err != nil || n == 0 {
			continue
		}
		for i := 0; i < n; i++ {
			text, err := scope.Nth(i).InnerText()
			if err != nil {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if _, ok := seen[text]; ok {
				continue
			}
			seen[text] = struct{}{}
			labels = append(labels, text)
		}
	}
	return labels
}

func (u *Uploader) isVideoUploadComplete(page playwright.Page) bool {
	for _, label := range readUploadProgressLabels(page) {
		if complete, _ := uploadProgressFromLabel(label); complete {
			return true
		}
	}
	for _, sel := range uploadCompleteProgressSelectors() {
		if visible(page, sel) {
			return true
		}
	}
	return false
}

func uploadCompleteProgressSelectors() []string {
	return []string{
		`ytcp-uploads-dialog ytcp-video-upload-progress >> text=/Upload complete/i`,
		`ytcp-uploads-dialog ytcp-video-upload-progress >> text=/Processing will begin shortly/i`,
		`ytcp-uploads-dialog ytcp-video-upload-progress >> text=/Processing up to HD/i`,
		`ytcp-video-upload-progress >> text=/Upload complete/i`,
		`ytcp-video-upload-progress >> text=/Processing will begin shortly/i`,
		`ytcp-video-upload-progress >> text=/Processing up to HD/i`,
	}
}

func (u *Uploader) checkStudioUploadError(page playwright.Page) error {
	dialog := uploadDialog(page)
	loc := dialog.Locator(`.error-area.style-scope.ytcp-uploads-dialog`).First()
	n, err := loc.Count()
	if err != nil || n == 0 {
		loc = page.Locator(`.error-area.style-scope.ytcp-uploads-dialog`).First()
		n, err = loc.Count()
		if err != nil || n == 0 {
			return nil
		}
	}
	ok, err := loc.IsVisible()
	if err != nil || !ok {
		return nil
	}
	text, err := loc.InnerText()
	if err != nil || strings.TrimSpace(text) == "" {
		return nil
	}
	return fmt.Errorf("youtube studio upload error: %s", strings.TrimSpace(text))
}

func (u *Uploader) waitForVideoUploadComplete(page playwright.Page) error {
	deadline := time.Now().Add(u.UploadUIWait)
	for time.Now().Before(deadline) {
		if err := u.checkStudioUploadError(page); err != nil {
			return err
		}
		if u.isVideoUploadComplete(page) {
			return nil
		}
		page.WaitForTimeout(5000)
	}
	return fmt.Errorf("video file upload did not finish within %s (expected Upload complete or post-upload processing label)", u.UploadUIWait)
}
