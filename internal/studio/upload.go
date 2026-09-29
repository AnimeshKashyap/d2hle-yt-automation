package studio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/d2hle/yt-automation/internal/domain"
	"github.com/mxschmitt/playwright-go"
)

type UploadCallbacks struct {
	OnStep func(step string)
}

type Uploader struct {
	StudioURL     string
	NavTimeout    time.Duration
	ActionTimeout time.Duration
	UploadUIWait  time.Duration
	StepOpenStudio time.Duration
	StepUploadDialog time.Duration
	StepMetadata  time.Duration
	StepWizard    time.Duration
	StepPublish   time.Duration
	StepVerify    time.Duration
}

func (u *Uploader) Run(ctx context.Context, page playwright.Page, req domain.CreateJobRequest, cb UploadCallbacks) (UploadResult, error) {
	emit := func(step string) {
		if cb.OnStep != nil {
			cb.OnStep(step)
		}
	}

	emit("open_studio")
	var openRes DetectResult
	if err := runStep(ctx, "open_studio", u.StepOpenStudio, func(context.Context) error {
		res, err := DetectOnPage(page, u.StudioURL, u.StepOpenStudio)
		if err != nil {
			return err
		}
		openRes = res
		return nil
	}); err != nil {
		return UploadResult{}, &StepError{Step: "open_studio", Err: err}
	}
	switch openRes.Status {
	case domain.SessionActive:
	case domain.SessionNeedsUserAction:
		return UploadResult{}, &StepError{Step: "open_studio", Err: ErrNeedsAuth}
	default:
		return UploadResult{}, &StepError{Step: "open_studio", Err: ErrNeedsAuth}
	}

	emit("start_upload_dialog")
	if err := runStep(ctx, "start_upload_dialog", u.StepUploadDialog, func(context.Context) error {
		return u.openUploadDialog(page)
	}); err != nil {
		return UploadResult{}, &StepError{Step: "start_upload_dialog", Err: err}
	}

	emit("uploading_file")
	if err := runStep(ctx, "uploading_file", u.UploadUIWait, func(context.Context) error {
		return u.selectVideoFile(page, req.Video.LocalPath)
	}); err != nil {
		return UploadResult{}, &StepError{Step: "uploading_file", Err: err}
	}

	emit("metadata")
	if err := runStep(ctx, "metadata", u.StepMetadata, func(context.Context) error {
		return u.fillMetadata(page, req)
	}); err != nil {
		return UploadResult{}, &StepError{Step: "metadata", Err: err}
	}

	if req.ScheduleAt != nil {
		return UploadResult{}, &StepError{Step: "schedule", Err: ErrScheduleNotReady}
	}

	emit("wizard")
	if err := runStep(ctx, "wizard", u.StepWizard, func(context.Context) error {
		return u.advanceThroughWizard(page, req)
	}); err != nil {
		return UploadResult{}, &StepError{Step: "wizard", Err: err}
	}

	emit("waiting_upload")
	if err := runStep(ctx, "waiting_upload", u.UploadUIWait, func(context.Context) error {
		return u.waitForVideoUploadComplete(page)
	}); err != nil {
		return UploadResult{}, &StepError{Step: "waiting_upload", Err: err}
	}

	emit("publish")
	if err := runStep(ctx, "publish", u.StepPublish, func(context.Context) error {
		return u.publish(page, req.Visibility)
	}); err != nil {
		return UploadResult{}, &StepError{Step: "publish", Err: err}
	}

	emit("verify")
	var videoURL string
	if err := runStep(ctx, "verify", u.StepVerify, func(context.Context) error {
		url, err := u.verifyPublish(page)
		if err != nil {
			return err
		}
		videoURL = url
		return nil
	}); err != nil {
		return UploadResult{}, &StepError{Step: "verify", Err: err}
	}
	return UploadResult{VideoURL: videoURL}, nil
}

func (u *Uploader) openUploadDialog(page playwright.Page) error {
	if err := clickFirst(page, []string{
		`#create-icon`,
		`ytcp-button#create-icon`,
		`[aria-label="Create"]`,
	}, u.StepUploadDialog); err != nil {
		return err
	}
	short := u.StepUploadDialog / 3
	if short < 5*time.Second {
		short = 5 * time.Second
	}
	return clickFirst(page, []string{
		`tp-yt-paper-item:has-text("Upload videos")`,
		`ytcp-text-menu #text-item-0`,
		`[role="menuitem"]:has-text("Upload")`,
		`text=/Upload videos/i`,
	}, short)
}

func (u *Uploader) selectVideoFile(page playwright.Page, path string) error {
	ms := float64(minDuration(u.StepUploadDialog, u.UploadUIWait).Milliseconds())
	chooser, err := page.ExpectFileChooser(func() error {
		return clickFirst(page, []string{
			`text=/Select files/i`,
			`ytcp-button:has-text("Select files")`,
			`#select-files-button`,
			`input[type="file"]`,
		}, u.ActionTimeout)
	}, playwright.PageExpectFileChooserOptions{Timeout: &ms})
	if err != nil {
		// Fallback: set files directly on input
		loc := page.Locator(`input[type="file"]`).First()
		if err := loc.SetInputFiles([]string{path}); err != nil {
			return err
		}
	} else {
		if err := chooser.SetFiles([]string{path}); err != nil {
			return err
		}
	}
	return u.waitForUploadUI(page)
}

// waitForUploadUI waits until the upload wizard shows metadata fields. The file may still
// be uploading in the background (progress bar at the bottom of the dialog).
func (u *Uploader) waitForUploadUI(page playwright.Page) error {
	wait := minDuration(u.StepUploadDialog, u.UploadUIWait)
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if visible(page, `#title-textarea`) || visible(page, `ytcp-social-suggestions-textbox`) {
			return nil
		}
		page.WaitForTimeout(300)
	}
	ms := float64(u.ActionTimeout.Milliseconds())
	return page.Locator(`#title-textarea, ytcp-social-suggestions-textbox`).First().WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: &ms,
	})
}

func (u *Uploader) fillMetadata(page playwright.Page, req domain.CreateJobRequest) error {
	dialog := uploadDialog(page)
	if n, _ := dialog.Count(); n == 0 {
		dialog = page.Locator("ytcp-uploads-dialog").First()
	}

	if err := fillTextboxIn(dialog, []string{
		`#title-textarea #textbox`,
		`ytcp-social-suggestions-textbox[label="Title"] #textbox`,
	}, req.Title, u.ActionTimeout); err != nil {
		return err
	}

	if req.Description != "" {
		_ = fillTextboxIn(dialog, []string{
			`#description-textarea #textbox`,
			`ytcp-social-suggestions-textbox[label="Description"] #textbox`,
		}, req.Description, u.ActionTimeout)
	}

	if len(req.Tags) > 0 {
		_ = clickFirstIn(dialog, []string{
			`#toggle-button`,
			`ytcp-button#toggle-button`,
			`#details ytcp-button:has-text("Show more")`,
		}, u.ActionTimeout)
		tags := strings.Join(req.Tags, ", ")
		_ = fillTextboxIn(dialog, []string{
			`#tags-container #textbox`,
			`#tags-container ytcp-free-text-chip-input #textbox`,
			`ytcp-social-suggestions-textbox[label="Tags"] #textbox`,
		}, tags, u.ActionTimeout)
	}

	u.setMadeForKids(page, req.Audience.MadeForKids)

	if req.Thumbnail != nil && req.Thumbnail.LocalPath != "" {
		_ = u.uploadThumbnail(page, req.Thumbnail.LocalPath)
	}
	return nil
}

func (u *Uploader) setMadeForKids(page playwright.Page, madeForKids bool) {
	label := "No, it's not made for kids"
	if madeForKids {
		label = "Yes, it's made for kids"
	}
	ms := float64(u.ActionTimeout.Milliseconds())
	exact := true
	dialog := uploadDialog(page)
	if n, _ := dialog.Count(); n > 0 {
		radio := dialog.GetByRole("radio", playwright.LocatorGetByRoleOptions{Name: label, Exact: &exact})
		if err := radio.First().Click(playwright.LocatorClickOptions{Timeout: &ms}); err == nil {
			return
		}
	}
	_ = page.GetByRole("radio", playwright.PageGetByRoleOptions{Name: label, Exact: &exact}).First().
		Click(playwright.LocatorClickOptions{Timeout: &ms})
}

func (u *Uploader) uploadThumbnail(page playwright.Page, path string) error {
	ms := float64(u.ActionTimeout.Milliseconds())
	chooser, err := page.ExpectFileChooser(func() error {
		return clickFirst(page, []string{
			`#file-picker`,
			`text=/Upload thumbnail/i`,
			`ytcp-thumbnail-editor`,
		}, u.ActionTimeout)
	}, playwright.PageExpectFileChooserOptions{Timeout: &ms})
	if err != nil {
		return err
	}
	return chooser.SetFiles([]string{path})
}

func (u *Uploader) advanceThroughWizard(page playwright.Page, req domain.CreateJobRequest) error {
	// Prefer wizard tabs over footer "Next" (page also has carousel "Next item" buttons).
	for _, tabName := range []string{"Video elements", "Checks", "Visibility"} {
		if err := u.selectWizardTab(page, tabName); err != nil {
			if err := u.clickWizardNext(page); err != nil {
				return err
			}
		}
		page.WaitForTimeout(400)
		if tabName == "Visibility" && isVisibilityStep(page) {
			return u.setVisibility(page, req.Visibility)
		}
	}
	if isVisibilityStep(page) {
		return u.setVisibility(page, req.Visibility)
	}
	return ErrStudioState
}

func (u *Uploader) selectWizardTab(page playwright.Page, name string) error {
	dismissCoachMarks(page)
	ms := float64(u.ActionTimeout.Milliseconds())
	exact := true
	dialog := uploadDialog(page)
	if n, _ := dialog.Count(); n > 0 {
		tab := dialog.GetByRole("tab", playwright.LocatorGetByRoleOptions{
			Name:  name,
			Exact: &exact,
		})
		if err := tab.First().Click(playwright.LocatorClickOptions{Timeout: &ms}); err == nil {
			return nil
		}
	}
	tab := page.GetByRole("tab", playwright.PageGetByRoleOptions{Name: name, Exact: &exact})
	return tab.First().Click(playwright.LocatorClickOptions{Timeout: &ms})
}

func (u *Uploader) clickWizardNext(page playwright.Page) error {
	dismissCoachMarks(page)
	ms := float64(u.ActionTimeout.Milliseconds())
	exact := true

	dialog := uploadDialog(page)
	if n, _ := dialog.Count(); n > 0 {
		btn := dialog.GetByRole("button", playwright.LocatorGetByRoleOptions{
			Name:  "Next",
			Exact: &exact,
		})
		if err := btn.Click(playwright.LocatorClickOptions{Timeout: &ms}); err == nil {
			return nil
		}
	}

	if err := clickFirst(page, []string{
		`ytcp-uploads-dialog #next-button`,
		`ytcp-uploads-dialog ytcp-button#next-button`,
		`ytcp-uploads-dialog .dialog-buttons ytcp-button:has-text("Next")`,
	}, u.ActionTimeout); err == nil {
		return nil
	}

	btn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Next", Exact: &exact})
	return btn.First().Click(playwright.LocatorClickOptions{Timeout: &ms})
}

// isVisibilityStep is true only on the Visibility wizard tab, not on Details ("Saved as private" badge).
func isVisibilityStep(page playwright.Page) bool {
	if isWizardTabSelected(page, "Visibility") {
		return true
	}
	panel := page.Locator("ytcp-video-visibility").First()
	n, err := panel.Count()
	if err != nil || n == 0 {
		return false
	}
	ok, err := panel.IsVisible()
	if err != nil || !ok {
		return false
	}
	// Details editor must not be the active panel.
	if visible(page, `#title-textarea`) && !isWizardTabSelected(page, "Visibility") {
		return false
	}
	return visibilityRadiosVisible(page)
}

func isWizardTabSelected(page playwright.Page, name string) bool {
	tab := page.GetByRole("tab", playwright.PageGetByRoleOptions{Name: name}).First()
	n, err := tab.Count()
	if err != nil || n == 0 {
		return false
	}
	selected, err := tab.GetAttribute("aria-selected")
	return err == nil && selected == "true"
}

func visibilityRadiosVisible(page playwright.Page) bool {
	scope := page.Locator("ytcp-video-visibility")
	n, _ := scope.Count()
	if n == 0 {
		scope = page.Locator("#privacy-radios")
		n, _ = scope.Count()
	}
	if n == 0 {
		return false
	}
	for _, label := range []string{"Private", "Unlisted", "Public"} {
		r := scope.GetByRole("radio", playwright.LocatorGetByRoleOptions{Name: label}).First()
		if n, _ := r.Count(); n > 0 {
			if ok, _ := r.IsVisible(); ok {
				return true
			}
		}
	}
	return false
}

func (u *Uploader) setVisibility(page playwright.Page, v domain.Visibility) error {
	var label string
	switch v {
	case domain.VisibilityPrivate:
		label = "Private"
	case domain.VisibilityUnlisted:
		label = "Unlisted"
	case domain.VisibilityPublic:
		label = "Public"
	default:
		return fmt.Errorf("unknown visibility %q", v)
	}
	ms := float64(u.ActionTimeout.Milliseconds())

	panel := page.Locator("ytcp-video-visibility").First()
	if n, _ := panel.Count(); n > 0 {
		radio := panel.GetByRole("radio", playwright.LocatorGetByRoleOptions{Name: label}).First()
		if err := radio.Click(playwright.LocatorClickOptions{Timeout: &ms}); err == nil {
			return nil
		}
	}

	if err := page.GetByRole("radio", playwright.PageGetByRoleOptions{Name: label}).First().
		Click(playwright.LocatorClickOptions{Timeout: &ms}); err == nil {
		return nil
	}

	return clickFirst(page, []string{
		`ytcp-video-visibility tp-yt-paper-radio-button:has-text("` + label + `")`,
		`#privacy-radios tp-yt-paper-radio-button:has-text("` + label + `")`,
	}, u.ActionTimeout)
}

func (u *Uploader) publish(page playwright.Page, v domain.Visibility) error {
	dismissCoachMarks(page)
	ms := float64(u.ActionTimeout.Milliseconds())
	btn := "Publish"
	if v == domain.VisibilityPrivate {
		btn = "Save"
	}
	exact := true
	dialog := uploadDialog(page)
	if n, _ := dialog.Count(); n > 0 {
		if err := dialog.GetByRole("button", playwright.LocatorGetByRoleOptions{Name: btn, Exact: &exact}).First().
			Click(playwright.LocatorClickOptions{Timeout: &ms}); err == nil {
			return nil
		}
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: btn, Exact: &exact}).First().
		Click(playwright.LocatorClickOptions{Timeout: &ms}); err != nil {
		return page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Publish", Exact: &exact}).First().
			Click(playwright.LocatorClickOptions{Timeout: &ms})
	}
	return nil
}

